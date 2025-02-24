package rocsp

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jmhodges/clock"
	"github.com/letsencrypt/boulder/cmd"
	"github.com/letsencrypt/boulder/metrics"
)

func makeClient() (*WritingClient, clock.Clock) {
	CACertFile := "../test/redis-tls/minica.pem"
	CertFile := "../test/redis-tls/boulder/cert.pem"
	KeyFile := "../test/redis-tls/boulder/key.pem"
	tlsConfig := cmd.TLSConfig{
		CACertFile: &CACertFile,
		CertFile:   &CertFile,
		KeyFile:    &KeyFile,
	}
	tlsConfig2, err := tlsConfig.Load()
	if err != nil {
		panic(err)
	}

	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:     []string{"10.33.33.2:4218"},
		Username:  "unittest-rw",
		Password:  "824968fa490f4ecec1e52d5e34916bdb60d45f8d",
		TLSConfig: tlsConfig2,
	})
	clk := clock.NewFake()
	return NewWritingClient(rdb, 5*time.Second, clk, metrics.NoopRegisterer), clk
}

func TestSetAndGet(t *testing.T) {
	client, _ := makeClient()

	response, err := ioutil.ReadFile("testdata/ocsp.response")
	if err != nil {
		t.Fatal(err)
	}
	var shortIssuerID byte = 99
	err = client.StoreResponse(context.Background(), response, byte(shortIssuerID))
	if err != nil {
		t.Fatalf("storing response: %s", err)
	}

	serial := "ffaa13f9c34be80b8e2532b83afe063b59a6"
	resp2, err := client.GetResponse(context.Background(), serial)
	if err != nil {
		t.Fatalf("getting response: %s", err)
	}
	if !bytes.Equal(resp2, response) {
		t.Errorf("response written and response retrieved were not equal")
	}

	metadata, err := client.GetMetadata(context.Background(), serial)
	if err != nil {
		t.Fatalf("getting metadata: %s", err)
	}
	if metadata.ShortIssuerID != shortIssuerID {
		t.Errorf("expected shortIssuerID %d, got %d", shortIssuerID, metadata.ShortIssuerID)
	}
	expectedTime, err := time.Parse(time.RFC3339, "2021-10-25T20:00:00Z")
	if err != nil {
		t.Fatalf("failed to parse time: %s", err)
	}
	if metadata.ThisUpdate != expectedTime {
		t.Errorf("expected ThisUpdate %q, got %q", expectedTime, metadata.ThisUpdate)
	}
}
