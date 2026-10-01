package registry_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ebarimt "github.com/techpartners-asia/ebarimt-pos3-go/v2"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/posapi"
	"github.com/techpartners-asia/ebarimt-pos3-go/v2/registry"
)

func newClient(t *testing.T, h http.HandlerFunc, mod ...func(*registry.Config)) *registry.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	cfg := registry.Config{BaseURL: srv.URL, HTTPClient: srv.Client(), Timeout: 5 * time.Second}
	for _, m := range mod {
		m(&cfg)
	}
	return registry.New(cfg)
}

func reply(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func asError(t *testing.T, err error) *ebarimt.Error {
	t.Helper()
	var e *ebarimt.Error
	require.True(t, errors.As(err, &e), "want *ebarimt.Error, got %T: %v", err, err)
	return e
}

func TestEnvironmentHosts(t *testing.T) {
	assert.Equal(t, "https://api.ebarimt.mn", ebarimt.Production.RegistryURL())
	assert.Equal(t, "https://st-api.ebarimt.mn", ebarimt.Staging.RegistryURL())
	assert.Equal(t, "https://auth.itc.gov.mn/auth/realms/ITC/protocol/openid-connect/token", ebarimt.Production.TokenURL())
	assert.Equal(t, "https://st.auth.itc.gov.mn/auth/realms/Staging/protocol/openid-connect/token", ebarimt.Staging.TokenURL())
	assert.Equal(t, "https://service.itc.gov.mn", ebarimt.Production.ServiceURL())
	var zero ebarimt.Environment
	assert.Equal(t, ebarimt.Production, zero, "the zero value is Production")
}

func TestTaxpayer(t *testing.T) {
	ctx := context.Background()
	t.Run("found", func(t *testing.T) {
		c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/info/check/getInfo", r.URL.Path)
			assert.Equal(t, "37900846788", r.URL.Query().Get("tin"))
			_, _ = w.Write([]byte(`{"msg":"Амжилттай","status":200,"data":{"name":"ТЕСТИЙН ХЭРЭГЛЭГЧ 1","freeProject":true,"cityPayer":true,"vatPayer":true,"found":true,"vatpayerRegisteredDate":"2019-02-16","isGovernment":false}}`))
		})
		got, err := c.Taxpayer(ctx, "37900846788")
		require.NoError(t, err)
		assert.Equal(t, registry.Taxpayer{Name: "ТЕСТИЙН ХЭРЭГЛЭГЧ 1", VATPayer: true, CityTaxPayer: true, VATFreeProject: true, VATPayerRegisteredDate: "2019-02-16"}, got)
	})
	t.Run("found=false is ErrNotFound", func(t *testing.T) {
		c := newClient(t, reply(200, `{"msg":"Амжилттай","status":200,"data":{"found":false}}`))
		_, err := c.Taxpayer(ctx, "1")
		assert.ErrorIs(t, err, ebarimt.ErrNotFound)
		assert.Equal(t, ebarimt.NotFound, asError(t, err).Kind)
	})
	t.Run("null data is ErrNotFound", func(t *testing.T) {
		c := newClient(t, reply(200, `{"status":200,"data":null}`))
		_, err := c.Taxpayer(ctx, "1")
		assert.ErrorIs(t, err, ebarimt.ErrNotFound)
	})
	t.Run("envelope status other than 200 is Rejected with its own words", func(t *testing.T) {
		c := newClient(t, reply(200, `{"msg":"Алдаа","status":401,"data":null}`))
		_, err := c.Taxpayer(ctx, "1")
		e := asError(t, err)
		assert.Equal(t, ebarimt.Rejected, e.Kind)
		assert.Equal(t, "401", e.Status)
		assert.Equal(t, "Алдаа", e.Message)
		assert.NotErrorIs(t, err, ebarimt.ErrNotFound)
	})
	t.Run("http 500 with a success-looking body is an error", func(t *testing.T) {
		c := newClient(t, reply(500, `{"status":200,"data":{"found":true,"vatPayer":true}}`))
		_, err := c.Taxpayer(ctx, "1")
		assert.Equal(t, 500, asError(t, err).HTTPStatus)
	})
	t.Run("junk 2xx body", func(t *testing.T) {
		c := newClient(t, reply(200, `<html>login</html>`))
		_, err := c.Taxpayer(ctx, "1")
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "login")
	})
	t.Run("empty tin is Invalid and sends nothing", func(t *testing.T) {
		var calls atomic.Int32
		c := newClient(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })
		_, err := c.Taxpayer(ctx, "")
		assert.Equal(t, ebarimt.Invalid, asError(t, err).Kind)
		assert.Zero(t, calls.Load())
	})
}

func TestTINByRegisterNumber(t *testing.T) {
	ctx := context.Background()
	for name, body := range map[string]string{
		"data as a number": `{"msg":"ok","status":200,"data":61200064714}`,
		"data as a string": `{"msg":"ok","status":200,"data":"61200064714"}`,
	} {
		t.Run(name, func(t *testing.T) {
			c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/info/check/getTinInfo", r.URL.Path)
				assert.Equal(t, "ЮЮ00112233", r.URL.Query().Get("regNo"))
				_, _ = w.Write([]byte(body))
			})
			tin, err := c.TINByRegisterNumber(ctx, "ЮЮ00112233")
			require.NoError(t, err)
			assert.Equal(t, "61200064714", tin)
		})
	}
	t.Run("the live answer for an unknown register number is ErrNotFound", func(t *testing.T) {
		// Captured live: HTTP 200, envelope status 500, data null.
		c := newClient(t, reply(200, `{"msg":"Хэрэглэгчийн мэдээлэл олдсонгүй.","status":500,"data":null}`))
		_, err := c.TINByRegisterNumber(ctx, "ЮЮ00000000")
		assert.ErrorIs(t, err, ebarimt.ErrNotFound)
		e := asError(t, err)
		assert.Equal(t, "500", e.Status, "the envelope's own status is kept")
		assert.Equal(t, "Хэрэглэгчийн мэдээлэл олдсонгүй.", e.Message)
	})
	t.Run("status 200 with data 0 or null is ErrNotFound", func(t *testing.T) {
		for _, body := range []string{`{"status":200,"data":0}`, `{"status":200,"data":null}`, `{"status":200,"data":""}`} {
			c := newClient(t, reply(200, body))
			_, err := c.TINByRegisterNumber(ctx, "x")
			assert.ErrorIs(t, err, ebarimt.ErrNotFound, body)
		}
	})
	t.Run("an unexpected envelope status is Rejected, not NotFound", func(t *testing.T) {
		c := newClient(t, reply(200, `{"msg":"unavailable","status":503,"data":null}`))
		_, err := c.TINByRegisterNumber(ctx, "x")
		assert.Equal(t, ebarimt.Rejected, asError(t, err).Kind)
		assert.NotErrorIs(t, err, ebarimt.ErrNotFound)
	})
	t.Run("http 500 is not read as a TIN", func(t *testing.T) {
		c := newClient(t, reply(500, `{"status":200,"data":31654321554}`))
		tin, err := c.TINByRegisterNumber(ctx, "x")
		assert.Empty(t, tin)
		assert.Equal(t, 500, asError(t, err).HTTPStatus)
	})
}

func TestDistricts(t *testing.T) {
	ctx := context.Background()
	t.Run("array", func(t *testing.T) {
		c := newClient(t, reply(200, `{"status":200,"msg":"ok","data":[{"branchCode":"01","branchName":"Архангай","subBranchCode":"02","subBranchName":"Чулуут"}]}`))
		got, err := c.Districts(ctx)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "0102", got[0].Code())
		assert.Equal(t, "Чулуут", got[0].SubBranchName)
	})
	t.Run("the docs' example shows a single object", func(t *testing.T) {
		c := newClient(t, reply(200, `{"status":200,"msg":"Амжилттай","data":{"branchCode":"01","branchName":"Архангай","subBranchCode":"02","subBranchName":"Чулуут"}}`))
		got, err := c.Districts(ctx)
		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "0102", got[0].Code())
	})
}

func TestTaxReliefCodes(t *testing.T) {
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/receipt/receipt/getProductTaxCode", r.URL.Path)
		_, _ = w.Write([]byte(`{"msg":"Амжилттай","status":200,"data":[{"startDate":"2016-01-01","endDate":"2024-12-31","taxProductCode":"43401","taxProductName":"Трактор","taxTypeCode":2,"taxTypeName":"VAT_FREE"},{"startDate":"2020-01-01","taxProductCode":"502","taxProductName":"Экспорт","taxTypeCode":3,"taxTypeName":"VAT_ZERO"}]}`))
	})
	got, err := c.TaxReliefCodes(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, registry.TaxReliefCode{Code: "43401", Name: "Трактор", TaxType: posapi.TaxVATExempt, StartDate: "2016-01-01", EndDate: "2024-12-31"}, got[0])
	assert.Equal(t, posapi.TaxVATZeroRated, got[1].TaxType)
	assert.Empty(t, got[1].EndDate)
}

func TestProductClassification(t *testing.T) {
	var gotPath string
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[["0","Хөдөө аж ахуй"],["1","Хүдэр"]]`))
	})
	ctx := context.Background()
	got, err := c.ProductClassification(ctx)
	require.NoError(t, err)
	assert.Equal(t, "/api/info/check/barcode/v2", gotPath)
	assert.Equal(t, []registry.ClassificationNode{{Code: "0", Name: "Хөдөө аж ахуй"}, {Code: "1", Name: "Хүдэр"}}, got)

	_, err = c.ProductClassification(ctx, "0", "01")
	require.NoError(t, err)
	assert.Equal(t, "/api/info/check/barcode/v2/0/01", gotPath)

	_, err = c.ProductClassification(ctx, "0", "", "011")
	assert.Equal(t, ebarimt.Invalid, asError(t, err).Kind)
	_, err = c.ProductClassification(ctx, "1", "2", "3", "4", "5", "6", "7")
	assert.Equal(t, ebarimt.Invalid, asError(t, err).Kind)
	_, err = c.ProductClassification(ctx, "../x")
	assert.Equal(t, ebarimt.Invalid, asError(t, err).Kind)
}

func TestRegisteredBarcodes(t *testing.T) {
	ctx := context.Background()
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/info/check/barcode/all", r.URL.Path)
		q := r.URL.Query()
		assert.Equal(t, "2", q.Get("page"))
		assert.Equal(t, "200", q.Get("size"))
		assert.Equal(t, "2025-02-21", q.Get("date"))
		_, _ = w.Write([]byte(`{"content":[["4007675393303","TSV Dark Berry 330ml ","2025-02-21","2413191",["stockQR"]],["8654001310334","Содон","2025-02-21","2143991",[]]],"number":2,"size":200,"totalPages":9,"totalElements":1700,"last":false}`))
	})
	got, err := c.RegisteredBarcodes(ctx, registry.Page{Number: 2, Size: 200, Since: time.Date(2025, 2, 21, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	assert.Equal(t, 9, got.TotalPages)
	assert.Equal(t, 1700, got.TotalElements)
	assert.False(t, got.Last)
	require.Len(t, got.Items, 2)
	assert.Equal(t, registry.Barcode{Barcode: "4007675393303", Name: "TSV Dark Berry 330ml ", ChangedOn: "2025-02-21", ClassificationCode: "2413191", ExciseStamp: true}, got.Items[0])
	assert.False(t, got.Items[1].ExciseStamp)

	for _, p := range []registry.Page{{Size: 0}, {Size: 201}, {Size: 10, Number: -1}} {
		_, err := c.RegisteredBarcodes(ctx, p)
		assert.Equal(t, ebarimt.Invalid, asError(t, err).Kind, "%+v", p)
	}
}

func TestMerchantLocations(t *testing.T) {
	ctx := context.Background()
	const body = `[{"mrchRegno":"99119911","registrationCode":1207127,"longitude":"-122.084","latitude":"37.42","storeName":"Test-1","address":null,"isCityTaxPayer":true,"licenses":[{"licenseNo":21930,"ownerRegno":"99119911","ownerName":"Тест","grantType":"ХОГ","grantApprveDate":"2026-05-25 10:03:53","grantTrminateDate":"2126-05-25 10:03:53"}]}]`
	var gotKey, gotQuery string
	mod := func(c *registry.Config) { c.APIKey = "KEY" }
	c := newClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotKey, gotQuery = r.Header.Get("X-API-KEY"), r.URL.RawQuery
		assert.Equal(t, "/api/info/cityTax/location", r.URL.Path)
		_, _ = w.Write([]byte(body))
	}, mod)
	got, err := c.MerchantLocations(ctx, "99119911", "101317077")
	require.NoError(t, err)
	assert.Equal(t, "KEY", gotKey)
	assert.Equal(t, "mrchRegno=99119911&posNo=101317077", gotQuery)
	require.Len(t, got, 1)
	assert.EqualValues(t, 1207127, got[0].RegistrationCode)
	assert.True(t, got[0].IsCityTaxPayer)
	assert.Empty(t, got[0].Address)
	require.Len(t, got[0].Licenses, 1)
	assert.EqualValues(t, 21930, got[0].Licenses[0].LicenseNo)

	t.Run("without an API key nothing is sent", func(t *testing.T) {
		var calls atomic.Int32
		c := newClient(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })
		_, err := c.MerchantLocations(ctx, "99119911", "1")
		assert.Equal(t, ebarimt.Invalid, asError(t, err).Kind)
		assert.Zero(t, calls.Load())
	})
	t.Run("a refused key is Rejected", func(t *testing.T) {
		c := newClient(t, reply(403, `{"status":403,"message":"Invalid API key"}`), mod)
		_, err := c.MerchantLocations(ctx, "99119911", "1")
		e := asError(t, err)
		assert.Equal(t, ebarimt.Rejected, e.Kind)
		assert.Equal(t, 403, e.HTTPStatus)
	})
}

func TestRetries(t *testing.T) {
	ctx := context.Background()
	flaky := func(failures int32) (http.HandlerFunc, *atomic.Int32) {
		var calls atomic.Int32
		return func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) <= failures {
				w.WriteHeader(http.StatusBadGateway)
				return
			}
			_, _ = w.Write([]byte(`{"status":200,"data":[]}`))
		}, &calls
	}
	fast := func(n int) func(*registry.Config) {
		return func(c *registry.Config) { c.Retries = n; c.RetryBackoff = time.Millisecond }
	}

	t.Run("off by default", func(t *testing.T) {
		h, calls := flaky(1)
		_, err := newClient(t, h).Districts(ctx)
		require.Error(t, err)
		assert.EqualValues(t, 1, calls.Load())
	})
	t.Run("retries a 5xx up to the limit", func(t *testing.T) {
		h, calls := flaky(2)
		_, err := newClient(t, h, fast(2)).Districts(ctx)
		require.NoError(t, err)
		assert.EqualValues(t, 3, calls.Load())

		h, calls = flaky(5)
		_, err = newClient(t, h, fast(2)).Districts(ctx)
		require.Error(t, err)
		assert.EqualValues(t, 3, calls.Load())
	})
	t.Run("does not retry a refusal", func(t *testing.T) {
		var calls atomic.Int32
		h := func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"status":"ERROR","message":"bad"}`))
		}
		_, err := newClient(t, h, fast(3)).Districts(ctx)
		require.Error(t, err)
		assert.EqualValues(t, 1, calls.Load())
	})
	t.Run("stops waiting when the context ends", func(t *testing.T) {
		h, _ := flaky(100)
		c := newClient(t, h, func(c *registry.Config) { c.Retries = 5; c.RetryBackoff = time.Hour })
		cctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()
		start := time.Now()
		_, err := c.Districts(cctx)
		require.Error(t, err)
		assert.Less(t, time.Since(start), 5*time.Second)
	})
}
