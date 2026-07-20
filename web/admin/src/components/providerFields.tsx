import type { ReactNode } from "react";

import { Badge, Field, PasswordInput, StringListField } from "./ui";

// Shared Google / Apple credential field groups (milestone 04), factored out
// of the Providers tab so the project-creation wizard (milestone 22) inlines
// the exact same fields with the same placeholders and write-only secret
// semantics. The tab passes its console walkthrough paragraphs through the
// `before*` slots; the wizard renders the bare fields, platform-aware.

// ---------- Google ----------

export type GoogleDraft = {
  webClientId: string;
  webClientSecret: string;
  iosClientId: string;
  androidClientId: string;
};

export const emptyGoogleDraft: GoogleDraft = {
  webClientId: "",
  webClientSecret: "",
  iosClientId: "",
  androidClientId: "",
};

export function GoogleCredentialFields({
  draft,
  onChange,
  showIos = true,
  showAndroid = true,
  hasStoredSecret = false,
  beforeWeb,
  beforeIos,
  beforeAndroid,
}: {
  draft: GoogleDraft;
  onChange: (draft: GoogleDraft) => void;
  showIos?: boolean;
  showAndroid?: boolean;
  hasStoredSecret?: boolean;
  beforeWeb?: ReactNode;
  beforeIos?: ReactNode;
  beforeAndroid?: ReactNode;
}) {
  const set = (patch: Partial<GoogleDraft>) => onChange({ ...draft, ...patch });
  return (
    <>
      {beforeWeb}
      <Field label="Web client ID">
        <input
          className="input input--mono"
          value={draft.webClientId}
          onChange={(e) => set({ webClientId: e.target.value })}
          placeholder="1234567890-abc.apps.googleusercontent.com"
          spellCheck={false}
        />
      </Field>
      <Field
        label="Web client secret"
        help={
          hasStoredSecret
            ? "A secret is stored (encrypted). Leave blank to keep it; paste a new value to replace it."
            : "Needed only for the web-redirect fallback flow. Stored encrypted, never shown again."
        }
      >
        <PasswordInput
          value={draft.webClientSecret}
          onChange={(v) => set({ webClientSecret: v })}
          placeholder={hasStoredSecret ? "•••••••• (stored)" : "GOCSPX-…"}
          autoComplete="off"
        />
      </Field>

      {showIos && (
        <>
          {beforeIos}
          <Field label="iOS client ID">
            <input
              className="input input--mono"
              value={draft.iosClientId}
              onChange={(e) => set({ iosClientId: e.target.value })}
              placeholder="1234567890-ios.apps.googleusercontent.com"
              spellCheck={false}
            />
          </Field>
        </>
      )}

      {showAndroid && (
        <>
          {beforeAndroid}
          <Field label="Android client ID">
            <input
              className="input input--mono"
              value={draft.androidClientId}
              onChange={(e) => set({ androidClientId: e.target.value })}
              placeholder="1234567890-android.apps.googleusercontent.com"
              spellCheck={false}
            />
          </Field>
        </>
      )}
    </>
  );
}

// ---------- Apple ----------

export type AppleDraft = {
  servicesId: string;
  teamId: string;
  keyId: string;
  privateKeyP8: string;
  bundleIds: string[];
};

export const emptyAppleDraft: AppleDraft = {
  servicesId: "",
  teamId: "",
  keyId: "",
  privateKeyP8: "",
  bundleIds: [],
};

// showNative gates the bundle-ID list (the native iOS flow's audiences);
// showWeb gates the Services ID + key trio the web/Android redirect fallback
// signs with. A web-only project never sees bundle IDs; an iOS-only project
// never sees the Services ID plumbing.
export function AppleCredentialFields({
  draft,
  onChange,
  showNative = true,
  showWeb = true,
  hasStoredKey = false,
  beforeBundleIds,
  beforeServicesId,
  beforeKey,
}: {
  draft: AppleDraft;
  onChange: (draft: AppleDraft) => void;
  showNative?: boolean;
  showWeb?: boolean;
  hasStoredKey?: boolean;
  beforeBundleIds?: ReactNode;
  beforeServicesId?: ReactNode;
  beforeKey?: ReactNode;
}) {
  const set = (patch: Partial<AppleDraft>) => onChange({ ...draft, ...patch });
  return (
    <>
      {showNative && (
        <>
          {beforeBundleIds}
          <StringListField
            label="Bundle IDs"
            values={draft.bundleIds}
            onChange={(v) => set({ bundleIds: v })}
            placeholder="com.example.birdwatch"
          />
        </>
      )}

      {showWeb && (
        <>
          {beforeServicesId}
          <Field label="Services ID">
            <input
              className="input input--mono"
              value={draft.servicesId}
              onChange={(e) => set({ servicesId: e.target.value })}
              placeholder="com.example.birdwatch.signin"
              spellCheck={false}
            />
          </Field>

          {beforeKey}
          <div className="row-16" style={{ alignItems: "flex-start" }}>
            <div style={{ flex: 1 }}>
              <Field label="Team ID">
                <input
                  className="input input--mono"
                  value={draft.teamId}
                  onChange={(e) => set({ teamId: e.target.value })}
                  placeholder="AB12CD34EF"
                  spellCheck={false}
                />
              </Field>
            </div>
            <div style={{ flex: 1 }}>
              <Field label="Key ID">
                <input
                  className="input input--mono"
                  value={draft.keyId}
                  onChange={(e) => set({ keyId: e.target.value })}
                  placeholder="XYZ987WV65"
                  spellCheck={false}
                />
              </Field>
            </div>
          </div>
          <Field
            label="Private key (.p8)"
            help={
              hasStoredKey
                ? "Leave blank to keep the stored key; paste a new one to replace it."
                : "Paste the full contents of the downloaded .p8 file. Stored encrypted, never shown again."
            }
          >
            <textarea
              className="input input--mono"
              rows={6}
              value={draft.privateKeyP8}
              onChange={(e) => set({ privateKeyP8: e.target.value })}
              placeholder={
                hasStoredKey
                  ? "Private key stored (encrypted)"
                  : "-----BEGIN PRIVATE KEY-----\n…\n-----END PRIVATE KEY-----"
              }
              spellCheck={false}
            />
          </Field>
          {hasStoredKey && <Badge tone="success">Private key stored</Badge>}
        </>
      )}
    </>
  );
}
