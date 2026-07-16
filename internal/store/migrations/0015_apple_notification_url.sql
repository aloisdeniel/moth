-- Milestone 12: persist the App Store Server Notification URL moth registered
-- for the app. Apple's App Store Connect API exposes no read for the configured
-- server-notification URL, so `moth setup billing` cannot tell whether it has
-- already registered its endpoint. Recording what moth last registered lets the
-- CLI stay idempotent — it re-registers (and reports a change) only when the URL
-- actually changes (e.g. the instance base URL moved), instead of every run.
-- Empty means moth has not registered a notification URL for this project.
ALTER TABLE billing_credentials
    ADD COLUMN apple_notification_url TEXT NOT NULL DEFAULT '';
