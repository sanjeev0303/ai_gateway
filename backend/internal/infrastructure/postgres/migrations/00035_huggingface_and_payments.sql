-- +goose Up
-- 1. Extend provider_kind CHECK constraints across provider_accounts, provider_health, models, pricing_models to support 'huggingface'

ALTER TABLE provider_accounts DROP CONSTRAINT IF EXISTS provider_accounts_provider_kind_check;
ALTER TABLE provider_accounts ADD CONSTRAINT provider_accounts_provider_kind_check
    CHECK (provider_kind IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom', 'huggingface'));

ALTER TABLE provider_health DROP CONSTRAINT IF EXISTS provider_health_provider_kind_check;
ALTER TABLE provider_health ADD CONSTRAINT provider_health_provider_kind_check
    CHECK (provider_kind IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom', 'huggingface'));

ALTER TABLE models DROP CONSTRAINT IF EXISTS models_provider_kind_check;
ALTER TABLE models ADD CONSTRAINT models_provider_kind_check
    CHECK (provider_kind IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom', 'huggingface'));

ALTER TABLE pricing_models DROP CONSTRAINT IF EXISTS pricing_models_provider_kind_check;
ALTER TABLE pricing_models ADD CONSTRAINT pricing_models_provider_kind_check
    CHECK (provider_kind IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom', 'huggingface'));

-- 2. Extend subscriptions to support dual payment providers (Stripe and Razorpay)
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS payment_provider text NOT NULL DEFAULT 'local'
    CHECK (payment_provider IN ('local', 'stripe', 'razorpay'));
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS provider_customer_id text;
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS provider_subscription_id text;

-- 3. Extend invoices to support paid settlement status and payment tracking
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_status_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_status_check
    CHECK (status IN ('open', 'void', 'paid'));

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS payment_provider text
    CHECK (payment_provider IS NULL OR payment_provider IN ('local', 'stripe', 'razorpay'));
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS payment_id text;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS paid_at timestamptz;

-- 4. Seed popular Hugging Face inference models into global catalog
INSERT INTO models (provider_kind, model_slug, display_name, is_active)
VALUES
    ('huggingface', 'meta-llama/Llama-3.3-70B-Instruct', 'Llama 3.3 70B Instruct', true),
    ('huggingface', 'mistralai/Mistral-7B-Instruct-v0.3', 'Mistral 7B Instruct v0.3', true),
    ('huggingface', 'Qwen/Qwen2.5-72B-Instruct', 'Qwen 2.5 72B Instruct', true),
    ('huggingface', 'deepseek-ai/DeepSeek-R1-Distill-Qwen-32B', 'DeepSeek R1 Distill Qwen 32B', true)
ON CONFLICT (provider_kind, model_slug) DO NOTHING;

-- +goose Down
DELETE FROM models WHERE provider_kind = 'huggingface';

ALTER TABLE invoices DROP COLUMN IF EXISTS paid_at;
ALTER TABLE invoices DROP COLUMN IF EXISTS payment_id;
ALTER TABLE invoices DROP COLUMN IF EXISTS payment_provider;
ALTER TABLE invoices DROP CONSTRAINT IF EXISTS invoices_status_check;
ALTER TABLE invoices ADD CONSTRAINT invoices_status_check
    CHECK (status IN ('open', 'void'));

ALTER TABLE subscriptions DROP COLUMN IF EXISTS provider_subscription_id;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS provider_customer_id;
ALTER TABLE subscriptions DROP COLUMN IF EXISTS payment_provider;

ALTER TABLE pricing_models DROP CONSTRAINT IF EXISTS pricing_models_provider_kind_check;
ALTER TABLE pricing_models ADD CONSTRAINT pricing_models_provider_kind_check
    CHECK (provider_kind IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom'));

ALTER TABLE models DROP CONSTRAINT IF EXISTS models_provider_kind_check;
ALTER TABLE models ADD CONSTRAINT models_provider_kind_check
    CHECK (provider_kind IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom'));

ALTER TABLE provider_health DROP CONSTRAINT IF EXISTS provider_health_provider_kind_check;
ALTER TABLE provider_health ADD CONSTRAINT provider_health_provider_kind_check
    CHECK (provider_kind IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom'));

ALTER TABLE provider_accounts DROP CONSTRAINT IF EXISTS provider_accounts_provider_kind_check;
ALTER TABLE provider_accounts ADD CONSTRAINT provider_accounts_provider_kind_check
    CHECK (provider_kind IN ('openai', 'anthropic', 'gemini', 'azure_openai', 'custom'));
