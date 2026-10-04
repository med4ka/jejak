package deeplink

import "testing"

// TestDetectEcommerce pins every platform branch plus the safety cases: a
// suffix-lookalike host must NOT match (the raw-Contains sketch would), and
// unparseable input must return the empty pair instead of a partial result.
func TestDetectEcommerce(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		platform string
		scheme   string
	}{
		{"shopee modern product", "https://shopee.co.id/product/111222333/444555666", "shopee", "shopeeid://product/111222333/444555666"},
		{"shopee legacy slug", "https://www.shopee.co.id/kaos-putih-i.555.666", "shopee", "shopeeid://product/555/666"},
		{"shopee non-product path", "https://shopee.co.id/fashion-men", "shopee", "shopeeid://"},
		{"shopee regional domain", "https://shopee.sg/product/1/2", "shopee", "shopeeid://product/1/2"},
		{"tokopedia product", "https://www.tokopedia.com/product/987654/123456", "tokopedia", "tokopedia://product/987654/123456"},
		{"tokopedia legacy slug", "https://tokopedia.com/toko-xyz/barang-i.10.20", "tokopedia", "tokopedia://product/10/20"},
		{"tokopedia non-product path", "https://www.tokopedia.com/toko-xyz", "tokopedia", "tokopedia://"},
		{"tiktok is web-only", "https://www.tiktok.com/@shop/video/7000000000000000000", "tiktok", ""},
		{"lazada root scheme", "https://www.lazada.co.id/products/i123456789.html", "lazada", "lazada://"},
		{"blibli root scheme", "https://www.blibli.com/some-product-pr-12345", "blibli", "blibli://"},
		{"bukalapak root scheme", "https://www.bukalapak.com/products/12345", "bukalapak", "bukalapak://"},
		{"ordinary link", "https://example.com/product/1/2", "", ""},
		{"suffix lookalike host", "https://shopee.co.id.evil.com/product/1/2", "", ""},
		{"lookalike subdomain", "https://tokopedia.com.evil.net/product/1/2", "", ""},
		{"javascript scheme", "javascript:alert(1)", "", ""},
		{"garbage input", "::::not-a-url::::", "", ""},
		{"empty string", "", "", ""},
		{"scheme relative", "//shopee.co.id/product/1/2", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			platform, scheme := DetectEcommerce(tc.in)
			if platform != tc.platform || scheme != tc.scheme {
				t.Errorf("DetectEcommerce(%q) = (%q, %q), want (%q, %q)",
					tc.in, platform, scheme, tc.platform, tc.scheme)
			}
		})
	}
}

// TestProductIDs guards the digit-only property: product IDs are re-embedded
// in an app scheme, so non-numeric segments must never be accepted.
func TestProductIDs(t *testing.T) {
	if ids, ok := productIDs("/product/12/34"); !ok || ids != "12/34" {
		t.Errorf("productIDs modern = (%q, %v), want (\"12/34\", true)", ids, ok)
	}
	if ids, ok := productIDs("/slug-i.7.8"); !ok || ids != "7/8" {
		t.Errorf("productIDs legacy = (%q, %v), want (\"7/8\", true)", ids, ok)
	}
	for _, bad := range []string{
		"/product/12/34/extra", // too many segments
		"/product/12ab/34",     // non-digit shop id
		"/products/12/34",      // wrong segment name
		"/product/12/",         // missing item id
		"/category/shoes",      // no product shape at all
	} {
		if ids, ok := productIDs(bad); ok {
			t.Errorf("productIDs(%q) accepted: ids=%q", bad, ids)
		}
	}
	// The legacy regexp matches PURE digit groups only: "-i.7x.8" must fail
	// because the shop-id group requires digits right after "-i.".
	if _, ok := productIDs("/slug-i.7x.8"); ok {
		t.Error(`productIDs("/slug-i.7x.8") accepted a non-digit shop id`)
	}
}
