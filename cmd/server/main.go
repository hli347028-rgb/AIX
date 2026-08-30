package main

import (
	"context"
	"flag"
	"os"
	"strings"
	"time"

	"backend/internal/conf"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/protobuf/types/known/durationpb"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z -X main.Name=backend"
var (
	// Name is the name of the compiled software.
	Name = "backend"
	// Version is the version of the compiled software.
	Version = "v1.0.0"
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

// defaultServerTimeout 兜底的单请求预算；kratos 自身默认仅 1s，对本服务偏紧。
const defaultServerTimeout = 5 * time.Second

// durationFromConfig 按路径读取形如 "5s" 的时长配置。
// 不能用 config.Value.Duration()：kratos 的实现是先按 int 解析再当纳秒，
// YAML 里的 "5s" 会解析失败并静默退回默认值。
func durationFromConfig(c config.Config, helper *log.Helper, key string, fallback time.Duration) time.Duration {
	raw, err := c.Value(key).String()
	if err != nil || strings.TrimSpace(raw) == "" {
		helper.Warnw("msg", "config missing, using fallback", "key", key, "fallback", fallback.String())
		return fallback
	}
	d, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil || d <= 0 {
		helper.Warnw("msg", "config invalid, using fallback", "key", key, "value", raw, "fallback", fallback.String())
		return fallback
	}
	return d
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	// 不依赖 Bootstrap protobuf 整包 Scan（server 段在部分环境下会扫成 nil）
	// 与 auth/wallet/database 一样按路径取值
	httpAddr, _ := c.Value("server.http.addr").String()
	grpcAddr, _ := c.Value("server.grpc.addr").String()
	if httpAddr == "" {
		httpAddr = "0.0.0.0:9000"
	}
	if grpcAddr == "" {
		grpcAddr = "0.0.0.0:9100"
	}
	helper := log.NewHelper(logger)
	httpTimeout := durationFromConfig(c, helper, "server.http.timeout", defaultServerTimeout)
	grpcTimeout := durationFromConfig(c, helper, "server.grpc.timeout", defaultServerTimeout)
	serverCfg := &conf.Server{
		Http: &conf.Server_HTTP{Addr: httpAddr, Timeout: durationpb.New(httpTimeout)},
		Grpc: &conf.Server_GRPC{Addr: grpcAddr, Timeout: durationpb.New(grpcTimeout)},
	}
	// 配置静默降级过一次（timeout 从未注入，实际跑在 kratos 默认 1s 上），
	// 这里把最终生效值打出来，便于一眼核对。
	helper.Infow(
		"msg", "server config resolved",
		"http.addr", httpAddr,
		"http.timeout", httpTimeout.String(),
		"grpc.addr", grpcAddr,
		"grpc.timeout", grpcTimeout.String(),
	)

	var authCfg conf.AuthConfig
	if err := c.Value("auth").Scan(&authCfg); err != nil {
		panic(err)
	}

	var walletCfg conf.WalletConfig
	if err := c.Value("wallet").Scan(&walletCfg); err != nil {
		panic(err)
	}

	var dbCfg conf.DatabaseConfig
	if err := c.Value("data.database").Scan(&dbCfg); err != nil {
		panic(err)
	}

	// 合作方转账加款接口配置。未配置时保持零值：没有任何合作方，
	// 接口对所有请求返回 1002，不影响其余功能启动。
	var partnerCfg conf.TransferPartnerConfig
	if err := c.Value("transfer_partners").Scan(&partnerCfg); err != nil {
		helper.Warnw("msg", "transfer_partners config missing or invalid, endpoint will reject all requests", "err", err)
	}
	helper.Infow("msg", "transfer partners loaded", "count", len(partnerCfg.Partners), "skew", partnerCfg.SkewDuration().String())

	app, settlementJob, chainRechargeJob, winPriceOracleJob, adminUsecase, cleanup, err := wireApp(serverCfg, &dbCfg, &authCfg, &walletCfg, &partnerCfg, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	_ = adminUsecase.LoadPersistedConfig(context.Background())

	settlementJob.Start()
	defer settlementJob.Stop()
	chainRechargeJob.Start()
	defer chainRechargeJob.Stop()
	winPriceOracleJob.Start()
	defer winPriceOracleJob.Stop()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
