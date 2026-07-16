import { useMutation, useQuery } from "@connectrpc/connect-query";
import { useEffect, useState } from "react";

import { errorMessage, invalidate } from "../api";
import {
  Badge,
  ConfirmDialog,
  Dialog,
  ErrorNote,
  Field,
  KeyWell,
  Loading,
  Status,
} from "../components/ui";
import {
  BillingCredentialsService,
  type AppleBillingConfig,
  type GoogleBillingConfig,
} from "../gen/moth/admin/v1/billing_credentials_pb";
import {
  EntitlementService,
  type Entitlement,
} from "../gen/moth/admin/v1/entitlement_pb";
import {
  MonetizationService,
  ProductSyncStatus,
  SyncAction,
  type GuidedStep,
  type ProductSyncItem,
  type StoreCatalogStatus,
} from "../gen/moth/admin/v1/monetization_pb";
import { ProductService, type Product } from "../gen/moth/admin/v1/product_pb";
import type { Project } from "../gen/moth/admin/v1/project_pb";
import { InstanceSettingsService } from "../gen/moth/admin/v1/settings_pb";
import { Store } from "../gen/moth/admin/v1/subscription_pb";
import { formatPrice, storeLabel } from "../lib/billing";
import { formatRelative } from "../lib/format";

// ProjectMonetization is the subscriptions & entitlements admin: the
// entitlement catalog apps gate on, the product tiers that grant them, and
// the store API credentials + webhook URLs that keep it all validated.
export function ProjectMonetization({ project }: { project: Project }) {
  return (
    <div className="stack-24" style={{ maxWidth: 820 }}>
      <p className="caption">
        moth mirrors each user's App Store / Google Play subscription and
        derives <strong>entitlements</strong> — the stable capability names
        (<span className="inline-code">pro</span>,{" "}
        <span className="inline-code">premium</span>) your app checks with{" "}
        <span className="inline-code">hasEntitlement()</span>. Apps gate on
        entitlements, never on a product id, so you can change which tier grants
        a capability without an app release. Declaring products is optional —
        every user always has a valid <span className="inline-code">none</span>{" "}
        (free) state.
      </p>

      <EntitlementsCard project={project} />
      <ProductsCard project={project} />
      <OfferingCard project={project} />
      <StoreConnectionCard project={project} />
      <BillingCredentialsCard project={project} />
    </div>
  );
}

// ---------- Entitlements ----------

function EntitlementsCard({ project }: { project: Project }) {
  const list = useQuery(EntitlementService.method.listEntitlements, {
    projectId: project.id,
  });
  const [editing, setEditing] = useState<Entitlement | "new">();
  const [removing, setRemoving] = useState<Entitlement>();

  const del = useMutation(EntitlementService.method.deleteEntitlement, {
    onSuccess: () => {
      invalidate(EntitlementService.method.listEntitlements);
      setRemoving(undefined);
    },
  });

  return (
    <section className="card card--pad stack-16">
      <div className="page__header">
        <h3 className="card__title">Entitlements</h3>
        <button type="button" className="btn btn--primary btn--compact" onClick={() => setEditing("new")}>
          Add entitlement
        </button>
      </div>
      <p className="caption">
        The named capabilities your app unlocks. Identifiers are immutable once
        created (apps depend on them); only the display name can change.
      </p>

      {list.isPending && <Loading />}
      {list.isError && <ErrorNote message={errorMessage(list.error)} />}
      {list.data &&
        (list.data.entitlements.length === 0 ? (
          <div className="empty">
            <p className="body-strong">No entitlements yet</p>
            <p className="caption">
              Add one (e.g. <span className="inline-code">pro</span>) to start
              gating features.
            </p>
          </div>
        ) : (
          <div style={{ overflowX: "auto" }}>
            <table className="table">
              <thead>
                <tr>
                  <th>Identifier</th>
                  <th>Display name</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {list.data.entitlements.map((e) => (
                  <tr key={e.id}>
                    <td className="mono">{e.identifier}</td>
                    <td>{e.displayName || <span className="text-tertiary">—</span>}</td>
                    <td>
                      <span className="row-8" style={{ justifyContent: "flex-end" }}>
                        <button
                          type="button"
                          className="btn btn--ghost btn--compact"
                          onClick={() => setEditing(e)}
                        >
                          Edit
                        </button>
                        <button
                          type="button"
                          className="btn btn--ghost btn--compact"
                          onClick={() => setRemoving(e)}
                        >
                          Delete
                        </button>
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ))}

      {editing && (
        <EntitlementDialog
          project={project}
          entitlement={editing === "new" ? undefined : editing}
          onClose={() => setEditing(undefined)}
        />
      )}
      <ConfirmDialog
        title="Delete entitlement"
        open={removing !== undefined}
        onClose={() => setRemoving(undefined)}
        onConfirm={() => removing && del.mutate({ projectId: project.id, id: removing.id })}
        confirmLabel="Delete entitlement"
        busy={del.isPending}
        error={del.isError ? errorMessage(del.error) : undefined}
      >
        <p>
          Deletes <span className="mono">{removing?.identifier}</span> and
          removes it from every product grant and operator grant. Apps checking
          it will read <span className="inline-code">false</span>.
        </p>
      </ConfirmDialog>
    </section>
  );
}

function EntitlementDialog({
  project,
  entitlement,
  onClose,
}: {
  project: Project;
  entitlement?: Entitlement;
  onClose: () => void;
}) {
  const editing = entitlement !== undefined;
  const [identifier, setIdentifier] = useState(entitlement?.identifier ?? "");
  const [displayName, setDisplayName] = useState(entitlement?.displayName ?? "");

  const done = {
    onSuccess: () => {
      invalidate(EntitlementService.method.listEntitlements);
      onClose();
    },
  };
  const create = useMutation(EntitlementService.method.createEntitlement, done);
  const update = useMutation(EntitlementService.method.updateEntitlement, done);
  const busy = create.isPending || update.isPending;
  const err = create.isError ? create.error : update.isError ? update.error : undefined;

  return (
    <Dialog title={editing ? "Edit entitlement" : "Add entitlement"} open onClose={onClose}>
      <form
        className="stack-16"
        onSubmit={(e) => {
          e.preventDefault();
          if (editing) {
            update.mutate({ projectId: project.id, id: entitlement.id, displayName });
          } else {
            create.mutate({ projectId: project.id, identifier: identifier.trim(), displayName });
          }
        }}
      >
        <Field
          label="Identifier"
          help={editing ? "The identifier is immutable." : "Lowercase, stable, e.g. \"pro\"."}
        >
          <input
            className="input input--mono"
            value={identifier}
            onChange={(e) => setIdentifier(e.target.value)}
            placeholder="pro"
            spellCheck={false}
            autoFocus={!editing}
            disabled={editing}
          />
        </Field>
        <Field label="Display name">
          <input
            className="input"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="Pro"
            autoFocus={editing}
          />
        </Field>
        {err && <p className="field__error">{errorMessage(err)}</p>}
        <div className="dialog__actions">
          <button type="button" className="btn btn--secondary" onClick={onClose}>
            Cancel
          </button>
          <button
            type="submit"
            className="btn btn--primary"
            disabled={busy || (!editing && identifier.trim() === "")}
          >
            {busy ? "Saving…" : editing ? "Save" : "Add entitlement"}
          </button>
        </div>
      </form>
    </Dialog>
  );
}

// ---------- Products ----------

function ProductsCard({ project }: { project: Project }) {
  const list = useQuery(ProductService.method.listProducts, { projectId: project.id });
  const ents = useQuery(EntitlementService.method.listEntitlements, { projectId: project.id });
  const [editing, setEditing] = useState<Product | "new">();
  const [removing, setRemoving] = useState<Product>();

  const entName = (id: string) =>
    ents.data?.entitlements.find((e) => e.id === id)?.identifier ?? id;

  const del = useMutation(ProductService.method.deleteProduct, {
    onSuccess: () => {
      invalidate(ProductService.method.listProducts);
      setRemoving(undefined);
    },
  });

  return (
    <section className="card card--pad stack-16">
      <div className="page__header">
        <h3 className="card__title">Products</h3>
        <button type="button" className="btn btn--primary btn--compact" onClick={() => setEditing("new")}>
          Add product
        </button>
      </div>
      <p className="caption">
        Your subscription tiers. Each maps to store SKUs and grants one or more
        entitlements while active. Products sharing an{" "}
        <span className="inline-code">offering</span> tag, in{" "}
        <span className="inline-code">sort order</span>, form one paywall.
        Price and period are display/analytics metadata — the store read stays
        authoritative.
      </p>

      {list.isPending && <Loading />}
      {list.isError && <ErrorNote message={errorMessage(list.error)} />}
      {list.data &&
        (list.data.products.length === 0 ? (
          <div className="empty">
            <p className="body-strong">No products yet</p>
            <p className="caption">Add a tier once you have at least one entitlement.</p>
          </div>
        ) : (
          <div style={{ overflowX: "auto" }}>
            <table className="table">
              <thead>
                <tr>
                  <th>Product</th>
                  <th>Grants</th>
                  <th>Price</th>
                  <th>Offering</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {list.data.products.map((p) => (
                  <tr key={p.id}>
                    <td>
                      <div className="stack-8" style={{ gap: 2 }}>
                        <span>{p.displayName || p.identifier}</span>
                        <span className="mono caption">{p.identifier}</span>
                      </div>
                    </td>
                    <td>
                      <span className="row-8" style={{ flexWrap: "wrap" }}>
                        {p.entitlementIds.length === 0 ? (
                          <span className="text-tertiary">—</span>
                        ) : (
                          p.entitlementIds.map((id) => <Badge key={id}>{entName(id)}</Badge>)
                        )}
                      </span>
                    </td>
                    <td className="mono">
                      {formatPrice(p.priceAmountMicros, p.currency)}
                      {p.billingPeriod && (
                        <span className="text-tertiary"> / {p.billingPeriod}</span>
                      )}
                    </td>
                    <td>
                      {p.offering ? (
                        <span className="mono">
                          {p.offering}
                          <span className="text-tertiary"> #{p.sortOrder}</span>
                        </span>
                      ) : (
                        <span className="text-tertiary">—</span>
                      )}
                    </td>
                    <td>
                      <span className="row-8" style={{ justifyContent: "flex-end" }}>
                        <button
                          type="button"
                          className="btn btn--ghost btn--compact"
                          onClick={() => setEditing(p)}
                        >
                          Edit
                        </button>
                        <button
                          type="button"
                          className="btn btn--ghost btn--compact"
                          onClick={() => setRemoving(p)}
                        >
                          Delete
                        </button>
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ))}

      {editing && (
        <ProductDialog
          project={project}
          product={editing === "new" ? undefined : editing}
          entitlements={ents.data?.entitlements ?? []}
          onClose={() => setEditing(undefined)}
        />
      )}
      <ConfirmDialog
        title="Delete product"
        open={removing !== undefined}
        onClose={() => setRemoving(undefined)}
        onConfirm={() => removing && del.mutate({ projectId: project.id, id: removing.id })}
        confirmLabel="Delete product"
        busy={del.isPending}
        error={del.isError ? errorMessage(del.error) : undefined}
      >
        <p>
          Deletes <span className="mono">{removing?.identifier}</span>. Existing
          subscriptions keep their store state but stop mapping to this tier.
        </p>
      </ConfirmDialog>
    </section>
  );
}

function ProductDialog({
  project,
  product,
  entitlements,
  onClose,
}: {
  project: Project;
  product?: Product;
  entitlements: Entitlement[];
  onClose: () => void;
}) {
  const editing = product !== undefined;
  const [identifier, setIdentifier] = useState(product?.identifier ?? "");
  const [displayName, setDisplayName] = useState(product?.displayName ?? "");
  const [appleId, setAppleId] = useState(product?.appleProductId ?? "");
  const [googleId, setGoogleId] = useState(product?.googleProductId ?? "");
  const [billingPeriod, setBillingPeriod] = useState(product?.billingPeriod ?? "");
  const [price, setPrice] = useState(
    product && product.priceAmountMicros !== 0n
      ? (Number(product.priceAmountMicros) / 1_000_000).toString()
      : "",
  );
  const [currency, setCurrency] = useState(product?.currency ?? "");
  const [trialPeriod, setTrialPeriod] = useState(product?.trialPeriod ?? "");
  const [offering, setOffering] = useState(product?.offering ?? "");
  const [sortOrder, setSortOrder] = useState(product?.sortOrder.toString() ?? "0");
  const [entIds, setEntIds] = useState<string[]>(product?.entitlementIds ?? []);

  const done = {
    onSuccess: () => {
      invalidate(ProductService.method.listProducts);
      onClose();
    },
  };
  const create = useMutation(ProductService.method.createProduct, done);
  const update = useMutation(ProductService.method.updateProduct, done);
  const busy = create.isPending || update.isPending;
  const err = create.isError ? create.error : update.isError ? update.error : undefined;

  function toggle(id: string) {
    setEntIds((cur) => (cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id]));
  }

  function submit() {
    const micros = price.trim() === "" ? 0n : BigInt(Math.round(parseFloat(price) * 1_000_000));
    const fields = {
      identifier: identifier.trim(),
      displayName: displayName.trim(),
      appleProductId: appleId.trim(),
      googleProductId: googleId.trim(),
      billingPeriod: billingPeriod.trim(),
      priceAmountMicros: micros,
      currency: currency.trim().toUpperCase(),
      trialPeriod: trialPeriod.trim(),
      offering: offering.trim(),
      sortOrder: parseInt(sortOrder, 10) || 0,
      entitlementIds: entIds,
    };
    if (editing) {
      update.mutate({ projectId: project.id, id: product.id, product: fields });
    } else {
      create.mutate({ projectId: project.id, product: fields });
    }
  }

  return (
    <Dialog title={editing ? "Edit product" : "Add product"} open onClose={onClose} wide>
      <form
        className="stack-16"
        onSubmit={(e) => {
          e.preventDefault();
          submit();
        }}
      >
        <div className="row-16" style={{ alignItems: "flex-start" }}>
          <div style={{ flex: 1 }}>
            <Field label="Identifier" help={editing ? "Immutable." : "e.g. \"monthly\""}>
              <input
                className="input input--mono"
                value={identifier}
                onChange={(e) => setIdentifier(e.target.value)}
                placeholder="monthly"
                spellCheck={false}
                disabled={editing}
                autoFocus={!editing}
              />
            </Field>
          </div>
          <div style={{ flex: 1 }}>
            <Field label="Display name">
              <input
                className="input"
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                placeholder="Monthly Pro"
              />
            </Field>
          </div>
        </div>

        <div className="row-16" style={{ alignItems: "flex-start" }}>
          <div style={{ flex: 1 }}>
            <Field label="Apple product id" help="App Store SKU; blank if Android-only.">
              <input
                className="input input--mono"
                value={appleId}
                onChange={(e) => setAppleId(e.target.value)}
                placeholder="com.example.pro.monthly"
                spellCheck={false}
              />
            </Field>
          </div>
          <div style={{ flex: 1 }}>
            <Field label="Google product id" help="Play SKU; blank if iOS-only.">
              <input
                className="input input--mono"
                value={googleId}
                onChange={(e) => setGoogleId(e.target.value)}
                placeholder="pro_monthly"
                spellCheck={false}
              />
            </Field>
          </div>
        </div>

        <div className="row-16" style={{ alignItems: "flex-start" }}>
          <div style={{ flex: 1 }}>
            <Field label="Price">
              <input
                className="input input--mono"
                value={price}
                onChange={(e) => setPrice(e.target.value)}
                placeholder="9.99"
                inputMode="decimal"
                spellCheck={false}
              />
            </Field>
          </div>
          <div style={{ flex: 1 }}>
            <Field label="Currency">
              <input
                className="input input--mono"
                value={currency}
                onChange={(e) => setCurrency(e.target.value)}
                placeholder="USD"
                spellCheck={false}
                maxLength={3}
              />
            </Field>
          </div>
          <div style={{ flex: 1 }}>
            <Field label="Billing period">
              <input
                className="input"
                value={billingPeriod}
                onChange={(e) => setBillingPeriod(e.target.value)}
                placeholder="monthly"
              />
            </Field>
          </div>
        </div>

        <Field label="Trial / intro descriptor" help={'Display-only, e.g. "7-day free trial".'}>
          <input
            className="input"
            value={trialPeriod}
            onChange={(e) => setTrialPeriod(e.target.value)}
            placeholder="7-day free trial"
          />
        </Field>

        <div className="stack-8">
          <span className="field__label">Grants entitlements</span>
          {entitlements.length === 0 ? (
            <p className="caption">
              Create an entitlement first — a product with no entitlement grants
              nothing.
            </p>
          ) : (
            entitlements.map((e) => (
              <label className="check" key={e.id}>
                <input
                  type="checkbox"
                  checked={entIds.includes(e.id)}
                  onChange={() => toggle(e.id)}
                />
                <span>
                  {e.displayName || e.identifier}{" "}
                  <span className="mono text-tertiary">{e.identifier}</span>
                </span>
              </label>
            ))
          )}
        </div>

        <div className="row-16" style={{ alignItems: "flex-start" }}>
          <div style={{ flex: 2 }}>
            <Field label="Offering" help="Paywall group tag; blank to omit.">
              <input
                className="input input--mono"
                value={offering}
                onChange={(e) => setOffering(e.target.value)}
                placeholder="default"
                spellCheck={false}
              />
            </Field>
          </div>
          <div style={{ flex: 1 }}>
            <Field label="Sort order">
              <input
                className="input input--mono"
                value={sortOrder}
                onChange={(e) => setSortOrder(e.target.value)}
                inputMode="numeric"
                spellCheck={false}
              />
            </Field>
          </div>
        </div>

        {err && <p className="field__error">{errorMessage(err)}</p>}
        <div className="dialog__actions">
          <button type="button" className="btn btn--secondary" onClick={onClose}>
            Cancel
          </button>
          <button
            type="submit"
            className="btn btn--primary"
            disabled={busy || identifier.trim() === ""}
          >
            {busy ? "Saving…" : editing ? "Save product" : "Add product"}
          </button>
        </div>
      </form>
    </Dialog>
  );
}

// ---------- Offering ----------

// OfferingCard orders the products a paywall presents. moth keeps one
// offering per project (the "default"); the first tier is the featured one
// most paywalls highlight. Reordering drives ReorderOffering, which persists
// the products' sort_order.
function OfferingCard({ project }: { project: Project }) {
  const offering = useQuery(MonetizationService.method.getOffering, {
    projectId: project.id,
    offering: "",
  });
  const reorder = useMutation(MonetizationService.method.reorderOffering, {
    onSuccess: () => {
      invalidate(
        MonetizationService.method.getOffering,
        ProductService.method.listProducts,
      );
    },
  });

  const products = offering.data?.offering?.products ?? [];

  function reorderTo(next: Product[]) {
    reorder.mutate({
      projectId: project.id,
      offering: "",
      productIds: next.map((p) => p.id),
    });
  }
  function move(index: number, dir: -1 | 1) {
    const j = index + dir;
    if (j < 0 || j >= products.length) return;
    const next = [...products];
    [next[index], next[j]] = [next[j], next[index]];
    reorderTo(next);
  }
  function feature(index: number) {
    if (index === 0) return;
    const next = [...products];
    const [p] = next.splice(index, 1);
    next.unshift(p);
    reorderTo(next);
  }

  return (
    <section className="card card--pad stack-16">
      <div className="page__header">
        <h3 className="card__title">Offering</h3>
        {reorder.isPending && <Badge>Saving…</Badge>}
      </div>
      <p className="caption">
        The ordered set of tiers your paywall presents. Drag order top-to-bottom;
        the <strong>featured</strong> tier sits first — the one a paywall
        highlights as recommended. Add a product to the offering by giving it an{" "}
        <span className="inline-code">offering</span> tag above.
      </p>

      {offering.isPending && <Loading />}
      {offering.isError && <ErrorNote message={errorMessage(offering.error)} />}
      {offering.data &&
        (products.length === 0 ? (
          <div className="empty">
            <p className="body-strong">No tiers in the offering</p>
            <p className="caption">
              Tag a product with an <span className="inline-code">offering</span>{" "}
              (e.g. <span className="inline-code">default</span>) to add it here.
            </p>
          </div>
        ) : (
          <div style={{ overflowX: "auto" }}>
            <table className="table">
              <thead>
                <tr>
                  <th style={{ width: 40 }}>#</th>
                  <th>Tier</th>
                  <th>Price</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {products.map((p, i) => (
                  <tr key={p.id}>
                    <td className="mono text-tertiary">{i + 1}</td>
                    <td>
                      <span className="row-8" style={{ flexWrap: "wrap" }}>
                        <span>{p.displayName || p.identifier}</span>
                        {i === 0 && <Badge tone="accent">Featured</Badge>}
                      </span>
                      <span className="mono caption">{p.identifier}</span>
                    </td>
                    <td className="mono">
                      {formatPrice(p.priceAmountMicros, p.currency)}
                      {p.billingPeriod && (
                        <span className="text-tertiary"> / {p.billingPeriod}</span>
                      )}
                    </td>
                    <td>
                      <span className="row-8" style={{ justifyContent: "flex-end" }}>
                        {i !== 0 && (
                          <button
                            type="button"
                            className="btn btn--ghost btn--compact"
                            disabled={reorder.isPending}
                            onClick={() => feature(i)}
                          >
                            Feature
                          </button>
                        )}
                        <button
                          type="button"
                          className="btn btn--ghost btn--compact"
                          aria-label={`Move ${p.identifier} up`}
                          disabled={reorder.isPending || i === 0}
                          onClick={() => move(i, -1)}
                        >
                          ↑
                        </button>
                        <button
                          type="button"
                          className="btn btn--ghost btn--compact"
                          aria-label={`Move ${p.identifier} down`}
                          disabled={reorder.isPending || i === products.length - 1}
                          onClick={() => move(i, 1)}
                        >
                          ↓
                        </button>
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ))}
      {reorder.isError && <ErrorNote message={errorMessage(reorder.error)} />}
    </section>
  );
}

// ---------- Store connection & catalog sync ----------

type Tone = "neutral" | "success" | "warning" | "danger" | "accent";

function actionMeta(a: SyncAction): { label: string; tone: Tone } {
  switch (a) {
    case SyncAction.CREATE:
      return { label: "Create", tone: "success" };
    case SyncAction.UPDATE:
      return { label: "Update", tone: "warning" };
    case SyncAction.NOOP:
      return { label: "In sync", tone: "neutral" };
    case SyncAction.GUIDED:
      return { label: "Manual", tone: "accent" };
    default:
      return { label: "—", tone: "neutral" };
  }
}

function syncStatusMeta(s: ProductSyncStatus): { label: string; tone: Tone } {
  switch (s) {
    case ProductSyncStatus.IN_SYNC:
      return { label: "In sync", tone: "success" };
    case ProductSyncStatus.DRIFT:
      return { label: "Drift", tone: "warning" };
    case ProductSyncStatus.ERROR:
      return { label: "Error", tone: "danger" };
    case ProductSyncStatus.PENDING:
      return { label: "Pending", tone: "neutral" };
    default:
      return { label: "—", tone: "neutral" };
  }
}

function StoreConnectionCard({ project }: { project: Project }) {
  const status = useQuery(MonetizationService.method.getStoreCatalogStatus, {
    projectId: project.id,
  });
  const byStore = (s: Store) => status.data?.stores.find((x) => x.store === s);

  return (
    <div className="stack-16">
      <div className="stack-8">
        <h3 className="card__title">Store connection</h3>
        <p className="caption">
          Whether each store's API credentials and renewal-notification plumbing
          are wired, and how moth's catalog compares to what's live in the store.
          Push moth's catalog to reconcile — moth automates what the store APIs
          allow and hands you the exact manual steps for the rest.
        </p>
      </div>
      {status.isError && <ErrorNote message={errorMessage(status.error)} />}
      {status.isPending ? (
        <Loading />
      ) : (
        <>
          <StoreStatusCard project={project} store={Store.APPLE} status={byStore(Store.APPLE)} />
          <StoreStatusCard project={project} store={Store.GOOGLE} status={byStore(Store.GOOGLE)} />
        </>
      )}
    </div>
  );
}

function StoreStatusCard({
  project,
  store,
  status,
}: {
  project: Project;
  store: Store;
  status?: StoreCatalogStatus;
}) {
  const [pushing, setPushing] = useState(false);
  const label = storeLabel(store);
  const apiName = store === Store.APPLE ? "App Store Connect" : "Google Play";
  const creds = status?.credentialsPresent ?? false;
  const notif = status?.notificationsWired ?? false;

  return (
    <section className="card card--pad stack-16">
      <div className="page__header">
        <h4 className="card__title">
          {label} — {apiName}
        </h4>
        {creds ? (
          <Badge tone="success">Connected</Badge>
        ) : (
          <Badge tone="warning">No credentials</Badge>
        )}
      </div>

      <div className="stack-8">
        <Status tone={creds ? "success" : "warning"}>
          {creds ? "API credentials configured" : "API credentials not configured"}
        </Status>
        <Status tone={notif ? "success" : "warning"}>
          {notif
            ? store === Store.APPLE
              ? "Server-notification URL registered"
              : "RTDN Pub/Sub topic wired"
            : "Renewal notifications not wired"}
        </Status>
      </div>

      <DiffSummary status={status} />

      <div className="row-12" style={{ justifyContent: "space-between", flexWrap: "wrap" }}>
        <span className="caption">
          {status?.lastSyncTime
            ? `Last synced ${formatRelative(status.lastSyncTime)}`
            : "Never synced"}
        </span>
        <button
          type="button"
          className="btn btn--secondary btn--compact"
          onClick={() => setPushing(true)}
        >
          Push to {label}
        </button>
      </div>

      {pushing && (
        <PushDialog project={project} store={store} onClose={() => setPushing(false)} />
      )}
    </section>
  );
}

function DiffSummary({ status }: { status?: StoreCatalogStatus }) {
  const total = status?.productsTotal ?? 0;
  if (total === 0) {
    return <p className="caption">No products mapped to this store yet.</p>;
  }
  return (
    <div className="row-8" style={{ flexWrap: "wrap" }}>
      <Badge>
        {total} product{total === 1 ? "" : "s"}
      </Badge>
      {status!.productsInSync > 0 && (
        <Badge tone="success">{status!.productsInSync} in sync</Badge>
      )}
      {status!.productsDrift > 0 && (
        <Badge tone="warning">{status!.productsDrift} drift</Badge>
      )}
      {status!.productsError > 0 && (
        <Badge tone="danger">{status!.productsError} error</Badge>
      )}
      {status!.productsUnmapped > 0 && (
        <Badge>{status!.productsUnmapped} unmapped</Badge>
      )}
    </div>
  );
}

// PushDialog runs a dry-run reconcile on open (the plan), then applies it on
// confirm. Both are idempotent by contract; a second push of an unchanged
// catalog reports in-sync with an all-noop plan.
function PushDialog({
  project,
  store,
  onClose,
}: {
  project: Project;
  store: Store;
  onClose: () => void;
}) {
  const label = storeLabel(store);
  const plan = useMutation(MonetizationService.method.syncStoreCatalog);
  const apply = useMutation(MonetizationService.method.syncStoreCatalog, {
    onSuccess: () => invalidate(MonetizationService.method.getStoreCatalogStatus),
  });

  useEffect(() => {
    plan.mutate({ projectId: project.id, store, dryRun: true });
    // Run the dry-run once when the dialog opens.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const applied = apply.data !== undefined;
  const result = apply.data ?? plan.data;

  return (
    <Dialog title={`Push catalog to ${label}`} open onClose={onClose} wide>
      <div className="stack-16">
        {plan.isPending && <Loading />}
        {plan.isError && <ErrorNote message={errorMessage(plan.error)} />}

        {result && !plan.isError && (
          <>
            {applied ? (
              <Status tone={result.items.some((i) => i.status === ProductSyncStatus.ERROR) ? "danger" : "success"}>
                {result.items.some((i) => i.status === ProductSyncStatus.ERROR)
                  ? "Pushed with errors — see below"
                  : "Catalog pushed"}
              </Status>
            ) : (
              <p className="caption">
                {result.inSync
                  ? `${label}'s catalog already matches moth — nothing to push.`
                  : "Review the plan, then push. moth changes only what drifted."}
              </p>
            )}

            {result.items.length === 0 && result.guidedSteps.length === 0 && (
              <div className="empty">
                <p className="body-strong">Nothing to reconcile</p>
                <p className="caption">
                  No products are mapped to {label}. Set a {label} product id on a
                  tier above first.
                </p>
              </div>
            )}

            {result.items.map((item) => (
              <SyncItemRow
                key={item.productId || item.identifier}
                item={item}
                applied={applied}
              />
            ))}

            {result.guidedSteps.length > 0 && (
              <div className="stack-8">
                <p className="body-strong">Manual steps</p>
                <p className="caption">
                  Steps the {label} API cannot perform — do these by hand with the
                  exact values shown.
                </p>
                {result.guidedSteps.map((step, i) => (
                  <GuidedStepView step={step} key={i} />
                ))}
              </div>
            )}
          </>
        )}

        {apply.isError && <ErrorNote message={errorMessage(apply.error)} />}

        <div className="dialog__actions">
          <button type="button" className="btn btn--secondary" onClick={onClose}>
            {applied ? "Close" : "Cancel"}
          </button>
          {!applied && (
            <button
              type="button"
              className="btn btn--primary"
              disabled={plan.isPending || plan.isError || apply.isPending || result?.inSync}
              onClick={() => apply.mutate({ projectId: project.id, store, dryRun: false })}
            >
              {apply.isPending ? "Pushing…" : `Push to ${label}`}
            </button>
          )}
        </div>
      </div>
    </Dialog>
  );
}

function SyncItemRow({ item, applied }: { item: ProductSyncItem; applied: boolean }) {
  const badge = applied ? syncStatusMeta(item.status) : actionMeta(item.action);
  return (
    <div className="sync-item stack-8">
      <div className="row-8" style={{ justifyContent: "space-between", flexWrap: "wrap" }}>
        <span className="mono">
          {item.identifier}
          {item.storeProductId && (
            <span className="text-tertiary"> · {item.storeProductId}</span>
          )}
        </span>
        <Badge tone={badge.tone}>{badge.label}</Badge>
      </div>
      {item.summary && <p className="caption">{item.summary}</p>}
      {item.changes.length > 0 && (
        <div className="stack-8">
          {item.changes.map((c) => (
            <div className="sync-change" key={c.field}>
              <span className="mono caption">{c.field}</span>
              <span className="mono caption">
                {c.current || "—"} → {c.desired || "—"}
              </span>
            </div>
          ))}
        </div>
      )}
      {item.error && <p className="field__error">{item.error}</p>}
      {item.guidedSteps.map((step, i) => (
        <GuidedStepView step={step} key={i} />
      ))}
    </div>
  );
}

function GuidedStepView({ step }: { step: GuidedStep }) {
  return (
    <div className="guided-step stack-8">
      <p className="body-strong">{step.title}</p>
      {step.detail && <p className="caption">{step.detail}</p>}
      {step.url && (
        <p className="caption">
          <a href={step.url} target="_blank" rel="noreferrer">
            {step.url}
          </a>
        </p>
      )}
      {step.values.map((v) => (
        <div className="stack-8" key={v.label}>
          <span className="field__label">{v.label}</span>
          <KeyWell value={v.value} />
        </div>
      ))}
    </div>
  );
}

// ---------- Billing credentials ----------

function BillingCredentialsCard({ project }: { project: Project }) {
  const creds = useQuery(BillingCredentialsService.method.getBillingCredentials, {
    projectId: project.id,
  });
  const instance = useQuery(InstanceSettingsService.method.getInstanceSettings);
  const base = instance.data?.baseUrl ?? "";

  if (creds.isPending) return <Loading />;
  if (creds.isError) return <ErrorNote message={errorMessage(creds.error)} />;

  return (
    <BillingCredentialsForm
      project={project}
      apple={creds.data.apple}
      google={creds.data.google}
      base={base}
    />
  );
}

function BillingCredentialsForm({
  project,
  apple,
  google,
  base,
}: {
  project: Project;
  apple?: AppleBillingConfig;
  google?: GoogleBillingConfig;
  base: string;
}) {
  // Apple
  const [iapKeyId, setIapKeyId] = useState(apple?.iapKeyId ?? "");
  const [iapIssuerId, setIapIssuerId] = useState(apple?.iapIssuerId ?? "");
  const [iapKeyP8, setIapKeyP8] = useState("");
  const [bundleId, setBundleId] = useState(apple?.bundleId ?? "");
  const [appAppleId, setAppAppleId] = useState(apple?.appAppleId ?? "");
  // Google
  const [serviceAccountJson, setServiceAccountJson] = useState("");
  const [packageName, setPackageName] = useState(google?.packageName ?? "");
  const [pubsubTopic, setPubsubTopic] = useState(google?.pubsubTopic ?? "");

  const [saved, setSaved] = useState(false);
  const update = useMutation(BillingCredentialsService.method.updateBillingCredentials, {
    onSuccess: () => {
      invalidate(BillingCredentialsService.method.getBillingCredentials);
      setIapKeyP8("");
      setServiceAccountJson("");
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    },
  });

  function save() {
    update.mutate({
      projectId: project.id,
      apple: {
        iapKeyId: iapKeyId.trim(),
        iapIssuerId: iapIssuerId.trim(),
        iapKeyP8: iapKeyP8.trim(),
        bundleId: bundleId.trim(),
        appAppleId: appAppleId.trim(),
        notificationSecret: "",
      },
      google: {
        serviceAccountJson: serviceAccountJson.trim(),
        packageName: packageName.trim(),
        pubsubTopic: pubsubTopic.trim(),
        rtdnSecret: "",
      },
    });
  }

  return (
    <form
      className="stack-24"
      onSubmit={(e) => {
        e.preventDefault();
        save();
      }}
    >
      <section className="card card--pad stack-16">
        <div className="page__header">
          <h3 className="card__title">Apple — App Store Server API</h3>
          {apple?.hasIapKey ? <Badge tone="success">Key stored</Badge> : <Badge>No key</Badge>}
        </div>
        <p className="caption">
          moth validates StoreKit 2 transactions and calls the App Store Server
          API with an <strong>In-App Purchase key</strong>. In{" "}
          <a
            href="https://appstoreconnect.apple.com/access/integrations/api"
            target="_blank"
            rel="noreferrer"
          >
            App Store Connect → Users and Access → Integrations → In-App Purchase
          </a>
          , generate a key, note its Key ID and the Issuer ID, and download the{" "}
          <span className="inline-code">.p8</span> (once).
        </p>

        <div className="row-16" style={{ alignItems: "flex-start" }}>
          <div style={{ flex: 1 }}>
            <Field label="Key ID">
              <input
                className="input input--mono"
                value={iapKeyId}
                onChange={(e) => setIapKeyId(e.target.value)}
                placeholder="2X9R4HXF34"
                spellCheck={false}
              />
            </Field>
          </div>
          <div style={{ flex: 1 }}>
            <Field label="Issuer ID">
              <input
                className="input input--mono"
                value={iapIssuerId}
                onChange={(e) => setIapIssuerId(e.target.value)}
                placeholder="57246542-96fe-1a63-e053-0824d011072a"
                spellCheck={false}
              />
            </Field>
          </div>
        </div>
        <Field
          label="In-App Purchase key (.p8)"
          help={
            apple?.hasIapKey
              ? "A key is stored (encrypted). Leave blank to keep it; paste a new one to replace it."
              : "Paste the full .p8 contents. Stored encrypted, never shown again."
          }
        >
          <textarea
            className="input input--mono"
            rows={5}
            value={iapKeyP8}
            onChange={(e) => setIapKeyP8(e.target.value)}
            placeholder={
              apple?.hasIapKey
                ? "Key stored (encrypted)"
                : "-----BEGIN PRIVATE KEY-----\n…\n-----END PRIVATE KEY-----"
            }
            spellCheck={false}
          />
        </Field>
        <div className="row-16" style={{ alignItems: "flex-start" }}>
          <div style={{ flex: 1 }}>
            <Field label="Bundle ID">
              <input
                className="input input--mono"
                value={bundleId}
                onChange={(e) => setBundleId(e.target.value)}
                placeholder="com.example.birdwatch"
                spellCheck={false}
              />
            </Field>
          </div>
          <div style={{ flex: 1 }}>
            <Field label="App Apple ID" help="Numeric App Store id.">
              <input
                className="input input--mono"
                value={appAppleId}
                onChange={(e) => setAppAppleId(e.target.value)}
                placeholder="1234567890"
                spellCheck={false}
              />
            </Field>
          </div>
        </div>
        <div className="stack-8">
          <span className="field__label">App Store Server Notifications URL</span>
          <p className="caption">
            Paste this as the Production and Sandbox notification URL (V2) in App
            Store Connect → your app → App Information.
          </p>
          {base && <KeyWell value={`${base}/billing/apple/notifications/${project.slug}`} />}
        </div>
      </section>

      <section className="card card--pad stack-16">
        <div className="page__header">
          <h3 className="card__title">Google — Play Developer API</h3>
          {google?.hasServiceAccount ? (
            <Badge tone="success">Service account stored</Badge>
          ) : (
            <Badge>No service account</Badge>
          )}
        </div>
        <p className="caption">
          moth resolves Play purchase tokens with a Google Cloud{" "}
          <strong>service account</strong> granted access in Play Console →
          Users and permissions, with the{" "}
          <a
            href="https://console.cloud.google.com/iam-admin/serviceaccounts"
            target="_blank"
            rel="noreferrer"
          >
            Google Cloud Console
          </a>{" "}
          JSON key. Real-time Developer Notifications arrive via a Cloud Pub/Sub
          topic delivered to the push URL below.
        </p>

        <Field
          label="Service account JSON"
          help={
            google?.hasServiceAccount
              ? "A service account is stored (encrypted). Leave blank to keep it; paste a new JSON to replace it."
              : "Paste the full downloaded JSON key. Stored encrypted, never shown again."
          }
        >
          <textarea
            className="input input--mono"
            rows={5}
            value={serviceAccountJson}
            onChange={(e) => setServiceAccountJson(e.target.value)}
            placeholder={
              google?.hasServiceAccount
                ? "Service account stored (encrypted)"
                : '{\n  "type": "service_account",\n  …\n}'
            }
            spellCheck={false}
          />
        </Field>
        <div className="row-16" style={{ alignItems: "flex-start" }}>
          <div style={{ flex: 1 }}>
            <Field label="Package name">
              <input
                className="input input--mono"
                value={packageName}
                onChange={(e) => setPackageName(e.target.value)}
                placeholder="com.example.birdwatch"
                spellCheck={false}
              />
            </Field>
          </div>
          <div style={{ flex: 1 }}>
            <Field label="Pub/Sub topic" help="The RTDN topic name.">
              <input
                className="input input--mono"
                value={pubsubTopic}
                onChange={(e) => setPubsubTopic(e.target.value)}
                placeholder="projects/my-gcp/topics/play-rtdn"
                spellCheck={false}
              />
            </Field>
          </div>
        </div>
        <div className="stack-8">
          <span className="field__label">RTDN Pub/Sub push URL</span>
          <p className="caption">
            Create a <strong>push</strong> subscription on the topic with this
            endpoint. moth reads the store on each nudge — it never trusts the
            notification body.
          </p>
          {base && <KeyWell value={`${base}/billing/google/rtdn/${project.slug}`} />}
        </div>
      </section>

      <div className="row-12">
        <button type="submit" className="btn btn--primary" disabled={update.isPending}>
          {update.isPending ? "Saving…" : "Save credentials"}
        </button>
        {saved && <span className="caption text-success">Saved.</span>}
        {update.isError && <span className="field__error">{errorMessage(update.error)}</span>}
      </div>
    </form>
  );
}
