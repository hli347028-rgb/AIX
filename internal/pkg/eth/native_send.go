package eth

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

// NativeTransferSendResult describes a broadcast native transfer.
type NativeTransferSendResult struct {
	TxHash      string
	Nonce       uint64
	FromAddress string
}

// TxReceiptOutcome summarizes an on-chain transaction receipt.
type TxReceiptOutcome struct {
	Found   bool
	Success bool
	Pending bool
}

// SendNativeTransfer signs and broadcasts a native coin transfer from the hot wallet.
func SendNativeTransfer(
	ctx context.Context,
	rpcURL, privateKeyHex, toAddress string,
	amount decimal.Decimal,
	decimals int32,
) (*NativeTransferSendResult, error) {
	return SendNativeTransferWithNonce(ctx, rpcURL, privateKeyHex, toAddress, amount, decimals, nil)
}

// SendNativeTransferWithNonce broadcasts using an explicit nonce when provided.
func SendNativeTransferWithNonce(
	ctx context.Context,
	rpcURL, privateKeyHex, toAddress string,
	amount decimal.Decimal,
	decimals int32,
	useNonce *uint64,
) (*NativeTransferSendResult, error) {
	rpcURL = strings.TrimSpace(rpcURL)
	toAddress = strings.TrimSpace(toAddress)
	privateKeyHex = strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	if rpcURL == "" {
		return nil, fmt.Errorf("rpc url not configured")
	}
	if privateKeyHex == "" {
		return nil, fmt.Errorf("withdraw private key not configured")
	}
	toNorm, err := NormalizeAddress(toAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid to address: %w", err)
	}

	rawAmount, err := amountToRaw(amount, decimals)
	if err != nil {
		return nil, err
	}
	if rawAmount.Sign() <= 0 {
		return nil, fmt.Errorf("transfer amount must be positive")
	}

	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	fromAddr := crypto.PubkeyToAddress(key.PublicKey)
	fromAddress := strings.ToLower(fromAddr.Hex())
	toAddr := common.HexToAddress(toNorm)

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("connect rpc failed: %w", err)
	}
	defer client.Close()

	var nonce uint64
	if useNonce != nil {
		nonce = *useNonce
	} else {
		nonce, err = client.PendingNonceAt(ctx, fromAddr)
		if err != nil {
			return nil, fmt.Errorf("get nonce failed: %w", err)
		}
	}

	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From: fromAddr, To: &toAddr, Value: rawAmount,
	})
	if err != nil {
		gasLimit = 21000
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("get gas price failed: %w", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get chain id failed: %w", err)
	}

	tx := types.NewTransaction(nonce, toAddr, rawAmount, gasLimit, gasPrice, nil)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return nil, fmt.Errorf("sign tx failed: %w", err)
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return nil, fmt.Errorf("send tx failed: %w", err)
	}
	return &NativeTransferSendResult{
		TxHash:      signed.Hash().Hex(),
		Nonce:       nonce,
		FromAddress: fromAddress,
	}, nil
}

// PendingNativeTransferNonce returns the next nonce for the hot wallet.
func PendingNativeTransferNonce(ctx context.Context, rpcURL, privateKeyHex string) (fromAddress string, nonce uint64, err error) {
	privateKeyHex = strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	if rpcURL == "" || privateKeyHex == "" {
		return "", 0, fmt.Errorf("rpc url or private key not configured")
	}
	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", 0, err
	}
	fromAddr := crypto.PubkeyToAddress(key.PublicKey)
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return "", 0, err
	}
	defer client.Close()
	nonce, err = client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return "", 0, err
	}
	return strings.ToLower(fromAddr.Hex()), nonce, nil
}

// InspectTransactionReceipt checks whether a tx is pending, successful, or failed.
func InspectTransactionReceipt(ctx context.Context, rpcURL, txHash string) (*TxReceiptOutcome, error) {
	txHash = strings.TrimSpace(txHash)
	if txHash == "" {
		return nil, fmt.Errorf("tx hash is empty")
	}
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	receipt, err := client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		return &TxReceiptOutcome{Pending: true}, nil
	}
	return &TxReceiptOutcome{
		Found:   true,
		Success: receipt.Status == types.ReceiptStatusSuccessful,
	}, nil
}

// FindTransactionBySenderNonce scans recent blocks for a mined tx with the given nonce.
func FindTransactionBySenderNonce(ctx context.Context, rpcURL, fromAddress string, nonce uint64, lookbackBlocks uint64) (txHash string, success bool, found bool, err error) {
	fromAddress, err = NormalizeAddress(fromAddress)
	if err != nil {
		return "", false, false, err
	}
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return "", false, false, err
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return "", false, false, err
	}
	signer := types.LatestSignerForChainID(chainID)
	fromAddr := common.HexToAddress(fromAddress)

	head, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return "", false, false, err
	}
	if lookbackBlocks == 0 {
		lookbackBlocks = 500
	}
	start := head.Number.Uint64()
	var end uint64
	if start > lookbackBlocks {
		end = start - lookbackBlocks
	}

	for number := start + 1; number > end; number-- {
		blockNum := number - 1
		block, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNum))
		if err != nil || block == nil {
			continue
		}
		for _, tx := range block.Transactions() {
			sender, err := types.Sender(signer, tx)
			if err != nil || sender != fromAddr || tx.Nonce() != nonce {
				continue
			}
			hash := tx.Hash().Hex()
			outcome, err := InspectTransactionReceipt(ctx, rpcURL, hash)
			if err != nil {
				return hash, false, true, nil
			}
			if outcome.Pending {
				return hash, false, true, nil
			}
			return hash, outcome.Success, true, nil
		}
	}
	return "", false, false, nil
}

// IsSenderNonceMined reports whether the sender has already mined the given nonce.
func IsSenderNonceMined(ctx context.Context, rpcURL, fromAddress string, nonce uint64) (bool, error) {
	fromAddress, err := NormalizeAddress(fromAddress)
	if err != nil {
		return false, err
	}
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return false, err
	}
	defer client.Close()
	latest, err := client.NonceAt(ctx, common.HexToAddress(fromAddress), nil)
	if err != nil {
		return false, err
	}
	return latest > nonce, nil
}
