package manager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

var (
	ErrJoinFailed = errors.New("failed to join cluster")
)

func normalizeAddr(addr string) string {
	if !strings.HasPrefix(addr, "http://") {
		return fmt.Sprintf("http://%s", addr)
	}

	return addr
}

func Join(srcIP string, joinAddr []string, id, addr string, numAttempts int,
	attemptInterval time.Duration) (string, error) {
	logger := zap.L().Named("manager-join")

	var j string
	var err error

	for i := 0; i < numAttempts; i++ {
		for _, a := range joinAddr {
			j, err = joinHelper(srcIP, a, id, addr, logger)
			if err == nil {
				return j, nil
			}
		}

		logger.Info("failed to join cluster", zap.Strings("join_addr", joinAddr), zap.Error(err))
		time.Sleep(attemptInterval)
	}

	logger.Info("failed to join cluster", zap.Strings("join_addr", joinAddr), zap.Int("num_attemps", numAttempts))
	return "", ErrJoinFailed
}

func joinHelper(src, joinAddr, id, addr string, logger *zap.Logger) (string, error) {
	if id == "" {
		return "", fmt.Errorf("note ID not set")
	}

	dialer := &net.Dialer{}

	if src != "" {
		dialer.LocalAddr = &net.TCPAddr{
			IP:   net.ParseIP(src),
			Port: 0,
		}
	}

	resolved, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return "", err
	}

	fulladdr := normalizeAddr(fmt.Sprintf("%s/join", joinAddr))

	transport := &http.Transport{Dial: dialer.Dial}
	client := &http.Client{Transport: transport}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	for {
		b, err := json.Marshal(map[string]interface{}{
			"id":   id,
			"addr": resolved.String(),
		})
		if err != nil {
			return "", err
		}

		resp, err := client.Post(fulladdr, "application-type/json", bytes.NewReader(b))
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		b, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		switch resp.StatusCode {
		case http.StatusOK:
			return fulladdr, nil
		case http.StatusMovedPermanently:
			fulladdr = resp.Header.Get("location")
			if fulladdr == "" {
				return "", fmt.Errorf("failed to join, invalid redirect received")
			}
			continue
		default:
			return "", fmt.Errorf("failed to join, node returned: %s: (%s)", resp.Status, string(b))
		}
	}
}
