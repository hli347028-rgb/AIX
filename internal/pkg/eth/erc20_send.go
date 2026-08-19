package eth

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
)

const erc20TransferABI = `[{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"value","type":"uint256"}],"name":"transfer","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"nonpayable","type":"function"}]`

// SendERC20Transfer signs and broadcasts an ERC20 transfer from the hot wallet.
func SendERC20Transfer(
	ctx context.Context,
	rpcURL, privateKeyHex, tokenContract, toAddress string,
	amount decimal.Decimal,
	decimals int32,
) (txHash string, err error) {
	rpcURL = strings.TrimSpace(rpcURL)
	tokenContract = strings.TrimSpace(tokenContract)
	toAddress = strings.TrimSpace(toAddress)
	privateKeyHex = strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	if rpcURL == "" {
		return "", fmt.Errorf("rpc url not configured")
	}
	if privateKeyHex == "" {
		return "", fmt.Errorf("withdraw private key not configured")
	}
	if tokenContract == "" {
		return "", fmt.Errorf("token contract not configured")
	}
	toNorm, err := NormalizeAddress(toAddress)
	if err != nil {
		return "", fmt.Errorf("invalid to address: %w", err)
	}

	rawAmount, err := amountToRaw(amount, decimals)
	if err != nil {
		return "", err
	}

	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid private key: %w", err)
	}
	fromAddr := crypto.PubkeyToAddress(key.PublicKey)

	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return "", fmt.Errorf("connect rpc failed: %w", err)
	}
	defer client.Close()

	parsedABI, err := abi.JSON(strings.NewReader(erc20TransferABI))
	if err != nil {
		return "", fmt.Errorf("parse erc20 abi failed: %w", err)
	}
	data, err := parsedABI.Pack("transfer", common.HexToAddress(toNorm), rawAmount)
	if err != nil {
		return "", fmt.Errorf("pack transfer data failed: %w", err)
	}

	tokenAddr := common.HexToAddress(tokenContract)
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From: fromAddr,
		To:   &tokenAddr,
		Data: data,
	})
	if err != nil {
		gasLimit = 120000
	}

	nonce, err := client.PendingNonceAt(ctx, fromAddr)
	if err != nil {
		return "", fmt.Errorf("get nonce failed: %w", err)
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("get gas price failed: %w", err)
	}

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("get chain id failed: %w", err)
	}

	tx := types.NewTransaction(nonce, tokenAddr, big.NewInt(0), gasLimit, gasPrice, data)
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
	if err != nil {
		return "", fmt.Errorf("sign tx failed: %w", err)
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return "", fmt.Errorf("send tx failed: %w", err)
	}
	return signed.Hash().Hex(), nil
}

// SenderAddressFromPrivateKey returns the checksummed address for a hex private key.
func SenderAddressFromPrivateKey(privateKeyHex string) (string, error) {
	privateKeyHex = strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	if privateKeyHex == "" {
		return "", fmt.Errorf("private key is empty")
	}
	key, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return "", err
	}
	return strings.ToLower(crypto.PubkeyToAddress(key.PublicKey).Hex()), nil
}
