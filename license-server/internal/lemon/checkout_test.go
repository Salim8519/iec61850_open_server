package lemon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateCheckoutSendsPriceInCents(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/checkouts" || r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("request %s %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"data":{"attributes":{"url":"https://s.lemonsqueezy.com/checkout/custom/abc"}}}`))
	}))
	defer server.Close()
	link, err := Client{Base: server.URL, APIKey: "k", StoreID: "7"}.CreateCheckout(context.Background(),
		CheckoutRequest{VariantID: "9", Price: "79.10", Custom: map[string]string{"hwid": "h"}})
	if err != nil || link != "https://s.lemonsqueezy.com/checkout/custom/abc" {
		t.Fatal(link, err)
	}
	attributes := got["data"].(map[string]any)["attributes"].(map[string]any)
	if attributes["custom_price"].(float64) != 7910 {
		t.Fatalf("custom_price %v", attributes["custom_price"])
	}
}

func TestVerifyAndParseOrder(t *testing.T) {
	body := []byte(`{"meta":{"event_name":"order_created","custom_data":{"hwid":"x"}},"data":{"type":"orders","id":"1",
		"attributes":{"identifier":"u","status":"paid","currency":"usd","subtotal":9905,"first_order_item":{"variant_id":3}}}}`)
	if Verify("s", body, "00") || Verify("", body, "") {
		t.Fatal("bad signatures must fail")
	}
	order, err := Parse(body)
	if err != nil || order.Subtotal != "99.05" || order.VariantID != "3" || order.Currency != "USD" || order.Custom["hwid"] != "x" {
		t.Fatalf("%+v %v", order, err)
	}
}
