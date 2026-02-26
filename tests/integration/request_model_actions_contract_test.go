package integration_test

import "testing"

func TestRequestModelActionsContract(t *testing.T) {
	t.Skip("Workfront 01 contract test: enable after request model/actions runtime is implemented")

	// Contract target behavior for /dashboard:
	//
	// 1) /actions-return-null: POST -> 204 no content.
	// 2) /actions-return-json: POST -> JSON response.
	// 3) /actions-redirect: POST -> redirect response.
	// 4) /get-vs-ssr: GET + non-HTML Accept -> GET handler.
	// 5) Payload/error mapping:
	//    - unsupported content type -> 415 envelope
	//    - invalid JSON -> 400 envelope
	//    - oversized payload (> 1 MiB default) -> 413 envelope
	// 6) HEAD auto-derived from GET.
	// 7) OPTIONS auto-generated with allowed methods.
	//
	// Keep this test as the contract source of truth while implementing:
	// parser -> codegen -> runtime dispatch -> context helpers.
}
