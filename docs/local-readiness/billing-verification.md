# Dual-Gateway Billing & Modes Verification

**Date**: 2026-10-19  
**Status**: **VERIFIED WITH DOCUMENTED LIMITATIONS**  
**Module**: [`backend/internal/application/billing`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/billing) & [`backend/internal/infrastructure/payment`](file:///home/sanju/ai-project/ai_gateway/backend/internal/infrastructure/payment)

---

## 1. Overview & Architecture

The billing subsystem supports dual payment gateways (Stripe and Razorpay) across three explicitly enforced operating modes:
1. **Disabled (`disabled`)**: Payment and checkout routes return HTTP 403 (`billing_disabled`). No payment interactions allowed.
2. **Simulation (`simulation`)**: Deterministic local test mode. Generates simulation URLs (`/checkout/simulate?session=...`); simulated webhook events do not grant real monetary entitlements; safe for automated testing.
3. **Provider Test Mode (`provider_test`)**: Uses live Stripe and Razorpay test credentials (`sk_test_...`, `rzp_test_...`); generates real hosted checkout sessions on provider test environments; requires authentic webhook HMAC signatures.

---

## 2. Billing Verification Matrix

| Area | Feature / Check | Verification Detail | Result |
| :--- | :--- | :--- | :--- |
| **Modes** | `disabled` mode safety | `POST /v1/billing/checkout` rejects with 403 `billing_disabled`. | **PASS** |
| **Modes** | `simulation` mode safety | Returns deterministic simulation URLs; prevents simulated events from granting real paid entitlements. | **PASS** |
| **Modes** | `provider_test` mode safety | Requires authentic test keys; rejects placeholder secrets (`sk_test_placeholder`); generates real provider checkout sessions. | **PASS** |
| **Stripe** | Checkout Session Creation | Uses `stripe.NewCustomGateway` with valid test keys; maps success/cancel URLs and customer email. | **PASS** |
| **Stripe** | Webhook HMAC Verification | Validates `Stripe-Signature` (`t=...,v1=...`) via constant-time HMAC-SHA256 comparison. | **PASS** |
| **Stripe** | Webhook Idempotency | Replay of identical `evt_...` returns HTTP 200 without duplicate credit grants. | **PASS** |
| **Razorpay** | Payment Link / Order Creation | Uses `razorpay.NewCustomGateway` with key ID and key secret; creates orders with amount in paise. | **PASS** |
| **Razorpay** | Webhook HMAC Verification | Validates `X-Razorpay-Signature` via HMAC-SHA256 using webhook secret. | **PASS** |
| **Razorpay** | Webhook Idempotency | Duplicate event processing is idempotent. | **PASS** |
| **Invoices** | Itemized Lines & Usage | Generates itemized invoices with token counts, micro-dollar cost arithmetic, and currency codes. | **PASS** |
| **RBAC** | Member vs Admin Access | Non-privileged `RoleMember` callers receive HTTP 403 on invoices, subscriptions, and billing usage. | **PASS** |

---

## 3. Discovered Defects & Fixes

### 3.1 Ambiguous Simulation vs Real Checkout URLs
- **Root Cause**: Earlier code could return simulated URLs even when configured with real test keys, leading to confusion between simulation and real provider interactions.
- **Fix**: Made billing mode selection explicit in [`backend/internal/application/billing/service.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/application/billing/service.go). Simulated URLs are strictly restricted to `simulation` mode; `provider_test` requires active payment gateway adapters.
- **Verification**: Tested in [`backend/internal/interfaces/controlapi/billing_modes_test.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/billing_modes_test.go).

### 3.2 Webhook Signature Bypass Vulnerability
- **Root Cause**: Missing signature headers or empty webhook secrets previously caused fallbacks or unverified event acceptance.
- **Fix**: Enforced mandatory HMAC signature verification in both Stripe and Razorpay webhook handlers in [`backend/internal/interfaces/controlapi/billing.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/billing.go); invalid or missing signatures return HTTP 400 immediately.
- **Verification**: Tested in [`backend/internal/interfaces/controlapi/billing_checkout_webhook_test.go`](file:///home/sanju/ai-project/ai_gateway/backend/internal/interfaces/controlapi/billing_checkout_webhook_test.go).

---

## 4. Tests Executed & Results

| Test Suite | Command | Exit Code | Outcome |
| :--- | :--- | :--- | :--- |
| **Billing Service Tests** | `go test -count=1 ./internal/application/billing/...` | `0` | **PASS** |
| **Stripe Adapter Tests** | `go test -count=1 ./internal/infrastructure/payment/stripe/...` | `0` | **PASS** |
| **Razorpay Adapter Tests** | `go test -count=1 ./internal/infrastructure/payment/razorpay/...` | `0` | **PASS** |
| **Billing Modes & Safety** | `go test -run TestBillingModes ./internal/interfaces/controlapi` | `0` | **PASS** |
| **Webhook & Idempotency** | `go test -run TestBillingCheckoutWebhooks ./internal/interfaces/controlapi` | `0` | **PASS** |

---

## 5. Remaining Limitations

- **Production Live Settlements**: Tests are conducted with user-provided Stripe and Razorpay test-mode keys (`sk_test_...`, `rzp_test_...`). Live credit card transactions and bank transfers require production provider merchant accounts and are not triggered in local test suites.
