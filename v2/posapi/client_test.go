package posapi_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ebarimt "github.com/techpartners-asia/ebarimt-pos3-go/v2"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/money"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/posapi"
)

type answer struct {
	status int
	body   string
}

// newClient serves every request through h.
func newClient(t *testing.T, h func(r *http.Request, body []byte) answer) (*posapi.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		a := h(r, b)
		if a.status == 0 {
			a.status = http.StatusOK
		}
		w.WriteHeader(a.status)
		_, _ = w.Write([]byte(a.body))
	}))
	t.Cleanup(srv.Close)
	return posapi.New(posapi.Config{Endpoint: srv.URL, HTTPClient: srv.Client(), Timeout: 5 * time.Second}), srv
}

func fixed(status int, body string) func(*http.Request, []byte) answer {
	return func(*http.Request, []byte) answer { return answer{status, body} }
}

func asError(t *testing.T, err error) *ebarimt.Error {
	t.Helper()
	var e *ebarimt.Error
	require.True(t, errors.As(err, &e), "want *ebarimt.Error, got %T: %v", err, err)
	return e
}

const okReceipt = `{"status":"SUCCESS","id":"BILL1","version":"3.2.44","totalAmount":2800,"totalVAT":250,"totalCityTax":50,"qrData":"Q","lottery":"AA1","date":"2026-05-09 13:44:46","easy":true,"posId":101317077,"receipts":[{"id":"SUB1","taxType":"VAT_ABLE"}],"type":"B2C_RECEIPT"}`

func TestIssueReceipt(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path sends POST /rest/receipt as json and maps the answer", func(t *testing.T) {
		c, _ := newClient(t, func(r *http.Request, b []byte) answer {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/rest/receipt", r.URL.Path)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Contains(t, string(b), `"merchantTin":"37900846788"`)
			assert.Contains(t, string(b), `"totalAmount":2800`)
			return answer{200, okReceipt}
		})
		got, err := c.IssueReceipt(ctx, posapi.ReceiptRequest{MerchantTin: "37900846788", TotalAmount: money.FromTugrik(2800)})
		require.NoError(t, err)
		assert.Equal(t, "BILL1", got.ID)
		assert.Equal(t, "SUB1", got.Receipts[0].ID)
		assert.Equal(t, posapi.TaxVATable, got.Receipts[0].TaxType)
		assert.Equal(t, "Q", got.QRData)
		assert.Equal(t, "AA1", got.Lottery)
		assert.Equal(t, "250", got.TotalVAT.String(), "totalVAT (docs spelling) decodes case-insensitively")
		assert.Equal(t, "50", got.TotalCityTax.String())
		assert.True(t, got.Easy)
		assert.EqualValues(t, 101317077, got.PosID)
		assert.True(t, got.Date.Equal(time.Date(2026, 5, 9, 13, 44, 46, 0, posapi.Ulaanbaatar())), got.Date.String())
	})

	t.Run("non-2xx with json message is Rejected", func(t *testing.T) {
		c, _ := newClient(t, fixed(400, `{"status":"ERROR","message":"classificationCode талбарын утга хоосон"}`))
		_, err := c.IssueReceipt(ctx, posapi.ReceiptRequest{})
		e := asError(t, err)
		assert.Equal(t, ebarimt.Rejected, e.Kind)
		assert.Equal(t, 400, e.HTTPStatus)
		assert.Equal(t, "ERROR", e.Status)
		assert.Equal(t, "classificationCode талбарын утга хоосон", e.Message)
	})
	t.Run("non-2xx with numeric status and msg key", func(t *testing.T) {
		c, _ := newClient(t, fixed(500, `{"status":500,"msg":"boom"}`))
		_, err := c.IssueReceipt(ctx, posapi.ReceiptRequest{})
		e := asError(t, err)
		assert.Equal(t, ebarimt.Rejected, e.Kind)
		assert.Equal(t, "500", e.Status)
		assert.Equal(t, "boom", e.Message)
	})
	t.Run("non-2xx without a document is Transport carrying the http status", func(t *testing.T) {
		c, _ := newClient(t, fixed(502, "<html>bad gateway</html>"))
		_, err := c.IssueReceipt(ctx, posapi.ReceiptRequest{})
		e := asError(t, err)
		assert.Equal(t, ebarimt.Transport, e.Kind, "a proxy page is not a PosAPI refusal, and not a lost answer either")
		assert.Equal(t, 502, e.HTTPStatus)
		assert.Empty(t, e.Message)
		assert.NotContains(t, err.Error(), "html")
	})
	t.Run("a non-2xx answer is an error even when its body looks like success", func(t *testing.T) {
		c, _ := newClient(t, fixed(500, okReceipt))
		got, err := c.IssueReceipt(ctx, posapi.ReceiptRequest{})
		assert.Equal(t, 500, asError(t, err).HTTPStatus)
		assert.Empty(t, got.ID)
	})
	t.Run("2xx status ERROR is Rejected", func(t *testing.T) {
		c, _ := newClient(t, fixed(200, `{"status":"ERROR","message":"x"}`))
		_, err := c.IssueReceipt(ctx, posapi.ReceiptRequest{})
		e := asError(t, err)
		assert.Equal(t, ebarimt.Rejected, e.Kind)
		assert.Equal(t, "x", e.Message)
		assert.Equal(t, "ERROR", e.Status)
		assert.Equal(t, 200, e.HTTPStatus)
	})
	t.Run("2xx status PAYMENT is Rejected", func(t *testing.T) {
		c, _ := newClient(t, fixed(200, `{"status":"PAYMENT","message":"y"}`))
		_, err := c.IssueReceipt(ctx, posapi.ReceiptRequest{})
		e := asError(t, err)
		assert.Equal(t, ebarimt.Rejected, e.Kind)
		assert.Equal(t, "y", e.Message)
	})
	t.Run("undecodable 2xx is an error, Indeterminate because the daemon accepted the request", func(t *testing.T) {
		for name, body := range map[string]string{"html": "<html>proxy login</html>", "empty": ""} {
			c, _ := newClient(t, fixed(200, body))
			got, err := c.IssueReceipt(ctx, posapi.ReceiptRequest{})
			require.Error(t, err, name)
			assert.Equal(t, ebarimt.Indeterminate, asError(t, err).Kind, name)
			assert.Empty(t, got.ID)
			assert.NotContains(t, err.Error(), "proxy login")
		}
	})
	t.Run("secrets never reach the error", func(t *testing.T) {
		for _, status := range []int{200, 500} {
			c, _ := newClient(t, fixed(status, `{"status":"ERROR","message":"m","qrData":"SECRET","lottery":"AA1"}`))
			_, err := c.IssueReceipt(ctx, posapi.ReceiptRequest{})
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "SECRET")
			assert.NotContains(t, err.Error(), "AA1")
		}
	})
	t.Run("an empty endpoint is Invalid", func(t *testing.T) {
		_, err := posapi.New(posapi.Config{}).IssueReceipt(ctx, posapi.ReceiptRequest{})
		assert.Equal(t, ebarimt.Invalid, asError(t, err).Kind)
	})
}

// A receipt is filed once. A failure before the request is fully written is
// Transport (safe to retry); a failure after it is Indeterminate (the receipt
// may exist: do not blind-retry).
func TestIssueReceipt_IndeterminateVersusTransport(t *testing.T) {
	ctx := context.Background()

	t.Run("connection dropped after the request was read is Indeterminate", func(t *testing.T) {
		var reads atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body) // the request is fully written and received
			reads.Add(1)
			conn, _, err := w.(http.Hijacker).Hijack()
			require.NoError(t, err)
			_ = conn.Close() // ... and the answer is lost
		}))
		defer srv.Close()
		c := posapi.New(posapi.Config{Endpoint: srv.URL, Timeout: 5 * time.Second})
		_, err := c.IssueReceipt(ctx, posapi.ReceiptRequest{MerchantTin: "1"})
		e := asError(t, err)
		assert.Equal(t, ebarimt.Indeterminate, e.Kind)
		assert.Error(t, e.Unwrap(), "the cause is kept")
		assert.EqualValues(t, 1, reads.Load(), "no automatic retry")
		assert.Contains(t, err.Error(), "may exist")
	})

	t.Run("timeout after the request was written is Indeterminate", func(t *testing.T) {
		release := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			select {
			case <-release:
			case <-r.Context().Done():
			}
		}))
		defer srv.Close()
		defer close(release)
		c := posapi.New(posapi.Config{Endpoint: srv.URL, Timeout: 150 * time.Millisecond})
		_, err := c.IssueReceipt(ctx, posapi.ReceiptRequest{MerchantTin: "1"})
		assert.Equal(t, ebarimt.Indeterminate, asError(t, err).Kind)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("connection refused is Transport, not Indeterminate", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		addr := l.Addr().String()
		require.NoError(t, l.Close()) // nothing listens there now
		c := posapi.New(posapi.Config{Endpoint: "http://" + addr, Timeout: 5 * time.Second})
		_, err = c.IssueReceipt(ctx, posapi.ReceiptRequest{MerchantTin: "1"})
		e := asError(t, err)
		assert.Equal(t, ebarimt.Transport, e.Kind)
		assert.Zero(t, e.HTTPStatus)
		assert.Equal(t, e.Unwrap().Error(), err.Error(), "Error() is exactly the cause's text")
	})

	t.Run("cancelled context before the call is Transport", func(t *testing.T) {
		c, _ := newClient(t, fixed(200, okReceipt))
		cctx, cancel := context.WithCancel(ctx)
		cancel()
		_, err := c.IssueReceipt(cctx, posapi.ReceiptRequest{})
		assert.Equal(t, ebarimt.Transport, asError(t, err).Kind)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("other calls never report Indeterminate", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(io.Discard, r.Body)
			conn, _, _ := w.(http.Hijacker).Hijack()
			_ = conn.Close()
		}))
		defer srv.Close()
		c := posapi.New(posapi.Config{Endpoint: srv.URL, Timeout: 5 * time.Second})
		assert.Equal(t, ebarimt.Transport, asError(t, c.ReturnReceipt(ctx, "i", time.Now())).Kind)
		assert.Equal(t, ebarimt.Transport, asError(t, c.SendToTaxAuthority(ctx)).Kind)
	})
}

func TestIssueReceipt_NeverRetries(t *testing.T) {
	var calls atomic.Int32
	c, _ := newClient(t, func(*http.Request, []byte) answer { calls.Add(1); return answer{503, "<html/>"} })
	_, err := c.IssueReceipt(context.Background(), posapi.ReceiptRequest{})
	require.Error(t, err)
	assert.EqualValues(t, 1, calls.Load())
}

func TestRedirectIsNotFollowed(t *testing.T) {
	var followed atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/receipt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/followed")
		w.WriteHeader(http.StatusFound)
	})
	mux.HandleFunc("/followed", func(w http.ResponseWriter, r *http.Request) {
		followed.Store(true)
		_, _ = w.Write([]byte(okReceipt))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	for name, cfg := range map[string]posapi.Config{
		"default client":  {Endpoint: srv.URL},
		"injected client": {Endpoint: srv.URL, HTTPClient: srv.Client()}, // follows redirects by itself
	} {
		t.Run(name, func(t *testing.T) {
			_, err := posapi.New(cfg).IssueReceipt(context.Background(), posapi.ReceiptRequest{})
			assert.Equal(t, http.StatusFound, asError(t, err).HTTPStatus)
			assert.False(t, followed.Load())
		})
	}
}

func TestReturnReceipt(t *testing.T) {
	ctx := context.Background()
	t.Run("sends DELETE with the body pinned by the golden", func(t *testing.T) {
		var gotMethod, gotPath, gotBody, gotCT string
		c, _ := newClient(t, func(r *http.Request, b []byte) answer {
			gotMethod, gotPath, gotBody, gotCT = r.Method, r.URL.Path, string(b), r.Header.Get("Content-Type")
			return answer{200, `{"status":200,"message":"ok"}`}
		})
		at := time.Date(2026, 9, 30, 10, 11, 12, 0, time.UTC)
		require.NoError(t, c.ReturnReceipt(ctx, "123456789012345678901234567890123", at))
		want, err := os.ReadFile(filepath.Join("testdata", "receipt_return.golden.json"))
		require.NoError(t, err)
		assert.Equal(t, string(want), gotBody)
		assert.Equal(t, http.MethodDelete, gotMethod)
		assert.Equal(t, "/rest/receipt", gotPath)
		assert.Equal(t, "application/json", gotCT)
	})
	t.Run("empty 2xx is success", func(t *testing.T) {
		c, _ := newClient(t, fixed(200, ""))
		require.NoError(t, c.ReturnReceipt(ctx, "i", time.Now()))
	})
	t.Run("non-2xx with message is Rejected", func(t *testing.T) {
		c, _ := newClient(t, fixed(400, `{"status":"ERROR","message":"already deleted"}`))
		e := asError(t, c.ReturnReceipt(ctx, "i", time.Now()))
		assert.Equal(t, ebarimt.Rejected, e.Kind)
		assert.Equal(t, 400, e.HTTPStatus)
		assert.Equal(t, "already deleted", e.Message)
	})
	t.Run("non-2xx html", func(t *testing.T) {
		c, _ := newClient(t, fixed(503, "<html/>"))
		e := asError(t, c.ReturnReceipt(ctx, "i", time.Now()))
		assert.Equal(t, 503, e.HTTPStatus)
		assert.Empty(t, e.Message)
	})
	t.Run("2xx html", func(t *testing.T) {
		c, _ := newClient(t, fixed(200, "<html>login</html>"))
		require.Error(t, c.ReturnReceipt(ctx, "i", time.Now()))
	})
	t.Run("2xx with ERROR status is Rejected", func(t *testing.T) {
		c, _ := newClient(t, fixed(200, `{"status":"ERROR","message":"m"}`))
		e := asError(t, c.ReturnReceipt(ctx, "i", time.Now()))
		assert.Equal(t, ebarimt.Rejected, e.Kind)
		assert.Equal(t, "m", e.Message)
	})
	t.Run("2xx with numeric failure status is Rejected", func(t *testing.T) {
		c, _ := newClient(t, fixed(200, `{"status":500,"message":"Баримт олдсонгүй"}`))
		e := asError(t, c.ReturnReceipt(ctx, "i", time.Now()))
		assert.Equal(t, ebarimt.Rejected, e.Kind)
		assert.Equal(t, "500", e.Status)
		assert.Equal(t, "Баримт олдсонгүй", e.Message)
	})
	t.Run("2xx with numeric 0 or 200 is success", func(t *testing.T) {
		for _, b := range []string{`{"status":0}`, `{"status":200,"message":"ok"}`, `{}`} {
			c, _ := newClient(t, fixed(200, b))
			require.NoError(t, c.ReturnReceipt(ctx, "i", time.Now()), b)
		}
	})
}

func TestSendToTaxAuthority(t *testing.T) {
	ctx := context.Background()
	t.Run("happy path", func(t *testing.T) {
		c, _ := newClient(t, func(r *http.Request, _ []byte) answer {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/rest/sendData", r.URL.Path)
			return answer{200, `{}`}
		})
		require.NoError(t, c.SendToTaxAuthority(ctx))
	})
	t.Run("non-2xx with message", func(t *testing.T) {
		c, _ := newClient(t, fixed(500, `{"status":"ERROR","message":"db locked"}`))
		e := asError(t, c.SendToTaxAuthority(ctx))
		assert.Equal(t, ebarimt.Rejected, e.Kind)
		assert.Equal(t, "db locked", e.Message)
	})
	t.Run("2xx html", func(t *testing.T) {
		c, _ := newClient(t, fixed(200, "<html>login</html>"))
		require.Error(t, c.SendToTaxAuthority(ctx))
	})
}

func TestStatus(t *testing.T) {
	const body = `{"operatorName":"Op","operatorTIN":"123","posId":5,"posNo":"10000001","lastSentDate":"2026-09-30 10:00:00","leftLotteries":42,
		"appInfo":{"applicationDir":"/a"},
		"merchants":[{"name":"M","tin":"37900846788","customers":[{"name":"C","tin":"61200064714","vatPayer":true}]}]}`
	c, _ := newClient(t, func(r *http.Request, _ []byte) answer {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/rest/info", r.URL.Path)
		return answer{200, body}
	})
	st, err := c.Status(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Op", st.OperatorName)
	assert.Equal(t, "123", st.OperatorTIN)
	assert.EqualValues(t, 5, st.PosID)
	assert.True(t, st.LastSentAt.Equal(time.Date(2026, 9, 30, 10, 0, 0, 0, posapi.Ulaanbaatar())), st.LastSentAt.String())
	assert.Equal(t, 42, st.LotteriesLeft)
	require.Len(t, st.Merchants, 1)
	require.Len(t, st.Merchants[0].Customers, 1)
	assert.True(t, st.Merchants[0].Customers[0].VATPayer)

	t.Run("non-2xx", func(t *testing.T) {
		c, _ := newClient(t, fixed(500, `{"message":"down"}`))
		_, err := c.Status(context.Background())
		assert.Equal(t, "down", asError(t, err).Message)
	})
	t.Run("unrecognised date layout leaves LastSentAt zero but still answers", func(t *testing.T) {
		c, _ := newClient(t, fixed(200, `{"lastSentDate":"yesterday","leftLotteries":1}`))
		st, err := c.Status(context.Background())
		require.NoError(t, err)
		assert.True(t, st.LastSentAt.IsZero())
		assert.Equal(t, 1, st.LotteriesLeft)
	})
}

func TestBankAccounts(t *testing.T) {
	c, _ := newClient(t, func(r *http.Request, _ []byte) answer {
		assert.Equal(t, "/rest/bankAccounts", r.URL.Path)
		assert.Equal(t, "37900846788", r.URL.Query().Get("tin"))
		return answer{200, `[{"id":7,"tin":"37900846788","bankAccountNo":"5000","bankAccountName":"N","bankId":9,"iBan":"MN12","bankName":"Khan"}]`}
	})
	got, err := c.BankAccounts(context.Background(), "37900846788")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, posapi.BankAccount{ID: 7, TIN: "37900846788", AccountNo: "5000", AccountName: "N", BankID: 9, IBan: "MN12", BankName: "Khan"}, got[0])

	_, err = c.BankAccounts(context.Background(), "")
	assert.Equal(t, ebarimt.Invalid, asError(t, err).Kind)
}
