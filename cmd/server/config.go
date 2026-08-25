package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strconv"
	"strings"
)

type config struct {
	addr      string
	dataDir   string
	selfcheck bool
}

func parseConfig(args []string, portEnv string) (config, error) {
	defaultAddr := "127.0.0.1:19081"
	if strings.TrimSpace(portEnv) != "" {
		port, err := strconv.Atoi(portEnv)
		if err != nil || port < 1 || port > 65535 {
			return config{}, errors.New("PORT 必须是 1 到 65535 的端口号")
		}
		defaultAddr = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	}
	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	addr := flags.String("addr", defaultAddr, "HTTP 监听地址")
	dataDir := flags.String("data", "./data", "本地持久化目录")
	selfcheck := flags.Bool("selfcheck", false, "执行有界 HTTP 闭环自检后退出")
	if err := flags.Parse(args); err != nil {
		return config{}, fmt.Errorf("解析参数: %w", err)
	}
	if flags.NArg() != 0 {
		return config{}, errors.New("存在未识别的位置参数")
	}
	if err := validateAddr(*addr); err != nil {
		return config{}, err
	}
	if strings.TrimSpace(*dataDir) == "" {
		return config{}, errors.New("数据目录不能为空")
	}
	return config{addr: *addr, dataDir: filepath.Clean(*dataDir), selfcheck: *selfcheck}, nil
}

func validateAddr(addr string) error {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("无效监听地址 %q: %w", addr, err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("监听端口必须是 1 到 65535 的数字")
	}
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return errors.New("监听地址必须使用回环主机，不能绑定 0.0.0.0 或外部地址")
	}
	return nil
}
