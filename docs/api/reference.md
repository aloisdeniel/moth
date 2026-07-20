# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [moth/admin/v1/session.proto](#moth_admin_v1_session-proto)
    - [Admin](#moth-admin-v1-Admin)
    - [GetCurrentAdminRequest](#moth-admin-v1-GetCurrentAdminRequest)
    - [GetCurrentAdminResponse](#moth-admin-v1-GetCurrentAdminResponse)
    - [LoginRequest](#moth-admin-v1-LoginRequest)
    - [LoginResponse](#moth-admin-v1-LoginResponse)
    - [LogoutRequest](#moth-admin-v1-LogoutRequest)
    - [LogoutResponse](#moth-admin-v1-LogoutResponse)
  
    - [SessionService](#moth-admin-v1-SessionService)
  
- [moth/admin/v1/account.proto](#moth_admin_v1_account-proto)
    - [AcceptAdminInviteRequest](#moth-admin-v1-AcceptAdminInviteRequest)
    - [AcceptAdminInviteResponse](#moth-admin-v1-AcceptAdminInviteResponse)
    - [AdminInvite](#moth-admin-v1-AdminInvite)
    - [ChangePasswordRequest](#moth-admin-v1-ChangePasswordRequest)
    - [ChangePasswordResponse](#moth-admin-v1-ChangePasswordResponse)
    - [CreatePersonalAccessTokenRequest](#moth-admin-v1-CreatePersonalAccessTokenRequest)
    - [CreatePersonalAccessTokenResponse](#moth-admin-v1-CreatePersonalAccessTokenResponse)
    - [InviteAdminRequest](#moth-admin-v1-InviteAdminRequest)
    - [InviteAdminResponse](#moth-admin-v1-InviteAdminResponse)
    - [ListAdminInvitesRequest](#moth-admin-v1-ListAdminInvitesRequest)
    - [ListAdminInvitesResponse](#moth-admin-v1-ListAdminInvitesResponse)
    - [ListAdminsRequest](#moth-admin-v1-ListAdminsRequest)
    - [ListAdminsResponse](#moth-admin-v1-ListAdminsResponse)
    - [ListPersonalAccessTokensRequest](#moth-admin-v1-ListPersonalAccessTokensRequest)
    - [ListPersonalAccessTokensResponse](#moth-admin-v1-ListPersonalAccessTokensResponse)
    - [PersonalAccessToken](#moth-admin-v1-PersonalAccessToken)
    - [RevokeAdminInviteRequest](#moth-admin-v1-RevokeAdminInviteRequest)
    - [RevokeAdminInviteResponse](#moth-admin-v1-RevokeAdminInviteResponse)
    - [RevokePersonalAccessTokenRequest](#moth-admin-v1-RevokePersonalAccessTokenRequest)
    - [RevokePersonalAccessTokenResponse](#moth-admin-v1-RevokePersonalAccessTokenResponse)
  
    - [AdminAccountService](#moth-admin-v1-AdminAccountService)
  
- [moth/admin/v1/analytics.proto](#moth_admin_v1_analytics-proto)
    - [CurrencyAmount](#moth-admin-v1-CurrencyAmount)
    - [DailyStat](#moth-admin-v1-DailyStat)
    - [Event](#moth-admin-v1-Event)
    - [GetStatsRequest](#moth-admin-v1-GetStatsRequest)
    - [GetStatsResponse](#moth-admin-v1-GetStatsResponse)
    - [GetSubscriptionStatsRequest](#moth-admin-v1-GetSubscriptionStatsRequest)
    - [GetSubscriptionStatsResponse](#moth-admin-v1-GetSubscriptionStatsResponse)
    - [ListRecentEventsRequest](#moth-admin-v1-ListRecentEventsRequest)
    - [ListRecentEventsResponse](#moth-admin-v1-ListRecentEventsResponse)
    - [PlatformBreakdown](#moth-admin-v1-PlatformBreakdown)
    - [ProviderBreakdown](#moth-admin-v1-ProviderBreakdown)
    - [RunRollupRequest](#moth-admin-v1-RunRollupRequest)
    - [RunRollupResponse](#moth-admin-v1-RunRollupResponse)
    - [StatTiles](#moth-admin-v1-StatTiles)
    - [SubscriptionMonthlyStat](#moth-admin-v1-SubscriptionMonthlyStat)
    - [SubscriptionStoreBreakdown](#moth-admin-v1-SubscriptionStoreBreakdown)
    - [SubscriptionTierBreakdown](#moth-admin-v1-SubscriptionTierBreakdown)
    - [SubscriptionTiles](#moth-admin-v1-SubscriptionTiles)
  
    - [Granularity](#moth-admin-v1-Granularity)
  
    - [AnalyticsService](#moth-admin-v1-AnalyticsService)
  
- [moth/admin/v1/audit.proto](#moth_admin_v1_audit-proto)
    - [AuditEntry](#moth-admin-v1-AuditEntry)
    - [ListAuditLogRequest](#moth-admin-v1-ListAuditLogRequest)
    - [ListAuditLogResponse](#moth-admin-v1-ListAuditLogResponse)
  
    - [AuditService](#moth-admin-v1-AuditService)
  
- [moth/admin/v1/billing_credentials.proto](#moth_admin_v1_billing_credentials-proto)
    - [AppleBillingConfig](#moth-admin-v1-AppleBillingConfig)
    - [GetBillingCredentialsRequest](#moth-admin-v1-GetBillingCredentialsRequest)
    - [GetBillingCredentialsResponse](#moth-admin-v1-GetBillingCredentialsResponse)
    - [GoogleBillingConfig](#moth-admin-v1-GoogleBillingConfig)
    - [StripeBillingConfig](#moth-admin-v1-StripeBillingConfig)
    - [UpdateBillingCredentialsRequest](#moth-admin-v1-UpdateBillingCredentialsRequest)
    - [UpdateBillingCredentialsResponse](#moth-admin-v1-UpdateBillingCredentialsResponse)
  
    - [BillingCredentialsService](#moth-admin-v1-BillingCredentialsService)
  
- [moth/admin/v1/copy.proto](#moth_admin_v1_copy-proto)
    - [CopyKey](#moth-admin-v1-CopyKey)
    - [CopyRevision](#moth-admin-v1-CopyRevision)
    - [GetProjectCopyRequest](#moth-admin-v1-GetProjectCopyRequest)
    - [GetProjectCopyResponse](#moth-admin-v1-GetProjectCopyResponse)
    - [ListCopyRevisionsRequest](#moth-admin-v1-ListCopyRevisionsRequest)
    - [ListCopyRevisionsResponse](#moth-admin-v1-ListCopyRevisionsResponse)
    - [ListLocalesRequest](#moth-admin-v1-ListLocalesRequest)
    - [ListLocalesResponse](#moth-admin-v1-ListLocalesResponse)
    - [Locale](#moth-admin-v1-Locale)
    - [ResetCopyRequest](#moth-admin-v1-ResetCopyRequest)
    - [ResetCopyResponse](#moth-admin-v1-ResetCopyResponse)
    - [RestoreCopyRevisionRequest](#moth-admin-v1-RestoreCopyRevisionRequest)
    - [RestoreCopyRevisionResponse](#moth-admin-v1-RestoreCopyRevisionResponse)
    - [UpdateProjectCopyRequest](#moth-admin-v1-UpdateProjectCopyRequest)
    - [UpdateProjectCopyRequest.ValuesEntry](#moth-admin-v1-UpdateProjectCopyRequest-ValuesEntry)
    - [UpdateProjectCopyResponse](#moth-admin-v1-UpdateProjectCopyResponse)
  
    - [CopyScreen](#moth-admin-v1-CopyScreen)
  
    - [CopyService](#moth-admin-v1-CopyService)
  
- [moth/admin/v1/entitlement.proto](#moth_admin_v1_entitlement-proto)
    - [CreateEntitlementRequest](#moth-admin-v1-CreateEntitlementRequest)
    - [CreateEntitlementResponse](#moth-admin-v1-CreateEntitlementResponse)
    - [DeleteEntitlementRequest](#moth-admin-v1-DeleteEntitlementRequest)
    - [DeleteEntitlementResponse](#moth-admin-v1-DeleteEntitlementResponse)
    - [Entitlement](#moth-admin-v1-Entitlement)
    - [ListEntitlementsRequest](#moth-admin-v1-ListEntitlementsRequest)
    - [ListEntitlementsResponse](#moth-admin-v1-ListEntitlementsResponse)
    - [UpdateEntitlementRequest](#moth-admin-v1-UpdateEntitlementRequest)
    - [UpdateEntitlementResponse](#moth-admin-v1-UpdateEntitlementResponse)
  
    - [EntitlementService](#moth-admin-v1-EntitlementService)
  
- [moth/admin/v1/product.proto](#moth_admin_v1_product-proto)
    - [CreateProductRequest](#moth-admin-v1-CreateProductRequest)
    - [CreateProductResponse](#moth-admin-v1-CreateProductResponse)
    - [DeleteProductRequest](#moth-admin-v1-DeleteProductRequest)
    - [DeleteProductResponse](#moth-admin-v1-DeleteProductResponse)
    - [GetProductRequest](#moth-admin-v1-GetProductRequest)
    - [GetProductResponse](#moth-admin-v1-GetProductResponse)
    - [ListProductsRequest](#moth-admin-v1-ListProductsRequest)
    - [ListProductsResponse](#moth-admin-v1-ListProductsResponse)
    - [Product](#moth-admin-v1-Product)
    - [UpdateProductRequest](#moth-admin-v1-UpdateProductRequest)
    - [UpdateProductResponse](#moth-admin-v1-UpdateProductResponse)
  
    - [ProductService](#moth-admin-v1-ProductService)
  
- [moth/admin/v1/subscription.proto](#moth_admin_v1_subscription-proto)
    - [GetUserSubscriptionRequest](#moth-admin-v1-GetUserSubscriptionRequest)
    - [GetUserSubscriptionResponse](#moth-admin-v1-GetUserSubscriptionResponse)
    - [Grant](#moth-admin-v1-Grant)
    - [GrantEntitlementRequest](#moth-admin-v1-GrantEntitlementRequest)
    - [GrantEntitlementResponse](#moth-admin-v1-GrantEntitlementResponse)
    - [ListUserSubscriptionsRequest](#moth-admin-v1-ListUserSubscriptionsRequest)
    - [ListUserSubscriptionsResponse](#moth-admin-v1-ListUserSubscriptionsResponse)
    - [RevokeGrantRequest](#moth-admin-v1-RevokeGrantRequest)
    - [RevokeGrantResponse](#moth-admin-v1-RevokeGrantResponse)
    - [Subscription](#moth-admin-v1-Subscription)
  
    - [Store](#moth-admin-v1-Store)
    - [SubscriptionStatus](#moth-admin-v1-SubscriptionStatus)
  
    - [SubscriptionService](#moth-admin-v1-SubscriptionService)
  
- [moth/admin/v1/monetization.proto](#moth_admin_v1_monetization-proto)
    - [GetOfferingRequest](#moth-admin-v1-GetOfferingRequest)
    - [GetOfferingResponse](#moth-admin-v1-GetOfferingResponse)
    - [GetStoreCatalogStatusRequest](#moth-admin-v1-GetStoreCatalogStatusRequest)
    - [GetStoreCatalogStatusResponse](#moth-admin-v1-GetStoreCatalogStatusResponse)
    - [GuidedStep](#moth-admin-v1-GuidedStep)
    - [GuidedValue](#moth-admin-v1-GuidedValue)
    - [Offering](#moth-admin-v1-Offering)
    - [ProductSyncItem](#moth-admin-v1-ProductSyncItem)
    - [ProductSyncState](#moth-admin-v1-ProductSyncState)
    - [ReorderOfferingRequest](#moth-admin-v1-ReorderOfferingRequest)
    - [ReorderOfferingResponse](#moth-admin-v1-ReorderOfferingResponse)
    - [StoreCatalogStatus](#moth-admin-v1-StoreCatalogStatus)
    - [SyncFieldChange](#moth-admin-v1-SyncFieldChange)
    - [SyncStoreCatalogRequest](#moth-admin-v1-SyncStoreCatalogRequest)
    - [SyncStoreCatalogResponse](#moth-admin-v1-SyncStoreCatalogResponse)
  
    - [ProductSyncStatus](#moth-admin-v1-ProductSyncStatus)
    - [SyncAction](#moth-admin-v1-SyncAction)
  
    - [MonetizationService](#moth-admin-v1-MonetizationService)
  
- [moth/admin/v1/paywall.proto](#moth_admin_v1_paywall-proto)
    - [GetPaywallConfigRequest](#moth-admin-v1-GetPaywallConfigRequest)
    - [GetPaywallConfigResponse](#moth-admin-v1-GetPaywallConfigResponse)
    - [ListPaywallRevisionsRequest](#moth-admin-v1-ListPaywallRevisionsRequest)
    - [ListPaywallRevisionsResponse](#moth-admin-v1-ListPaywallRevisionsResponse)
    - [PaywallConfig](#moth-admin-v1-PaywallConfig)
    - [PaywallLegal](#moth-admin-v1-PaywallLegal)
    - [PaywallRevision](#moth-admin-v1-PaywallRevision)
    - [ResetPaywallRequest](#moth-admin-v1-ResetPaywallRequest)
    - [ResetPaywallResponse](#moth-admin-v1-ResetPaywallResponse)
    - [RestorePaywallRevisionRequest](#moth-admin-v1-RestorePaywallRevisionRequest)
    - [RestorePaywallRevisionResponse](#moth-admin-v1-RestorePaywallRevisionResponse)
    - [UpdatePaywallConfigRequest](#moth-admin-v1-UpdatePaywallConfigRequest)
    - [UpdatePaywallConfigResponse](#moth-admin-v1-UpdatePaywallConfigResponse)
  
    - [PaywallLayout](#moth-admin-v1-PaywallLayout)
  
    - [PaywallService](#moth-admin-v1-PaywallService)
  
- [moth/admin/v1/theme.proto](#moth_admin_v1_theme-proto)
    - [DeleteLogoRequest](#moth-admin-v1-DeleteLogoRequest)
    - [DeleteLogoResponse](#moth-admin-v1-DeleteLogoResponse)
    - [GetThemeRequest](#moth-admin-v1-GetThemeRequest)
    - [GetThemeResponse](#moth-admin-v1-GetThemeResponse)
    - [ListThemeRevisionsRequest](#moth-admin-v1-ListThemeRevisionsRequest)
    - [ListThemeRevisionsResponse](#moth-admin-v1-ListThemeRevisionsResponse)
    - [ResetThemeRequest](#moth-admin-v1-ResetThemeRequest)
    - [ResetThemeResponse](#moth-admin-v1-ResetThemeResponse)
    - [RestoreThemeRevisionRequest](#moth-admin-v1-RestoreThemeRevisionRequest)
    - [RestoreThemeRevisionResponse](#moth-admin-v1-RestoreThemeRevisionResponse)
    - [Theme](#moth-admin-v1-Theme)
    - [ThemeColorOverrides](#moth-admin-v1-ThemeColorOverrides)
    - [ThemeColors](#moth-admin-v1-ThemeColors)
    - [ThemeLegal](#moth-admin-v1-ThemeLegal)
    - [ThemeLogo](#moth-admin-v1-ThemeLogo)
    - [ThemeRevision](#moth-admin-v1-ThemeRevision)
    - [ThemeShape](#moth-admin-v1-ThemeShape)
    - [ThemeSpacing](#moth-admin-v1-ThemeSpacing)
    - [ThemeTypography](#moth-admin-v1-ThemeTypography)
    - [UpdateThemeRequest](#moth-admin-v1-UpdateThemeRequest)
    - [UpdateThemeResponse](#moth-admin-v1-UpdateThemeResponse)
    - [UploadLogoRequest](#moth-admin-v1-UploadLogoRequest)
    - [UploadLogoResponse](#moth-admin-v1-UploadLogoResponse)
  
    - [LogoVariant](#moth-admin-v1-LogoVariant)
  
    - [ThemeService](#moth-admin-v1-ThemeService)
  
- [moth/admin/v1/project.proto](#moth_admin_v1_project-proto)
    - [AppleProviderConfig](#moth-admin-v1-AppleProviderConfig)
    - [CreateProjectRequest](#moth-admin-v1-CreateProjectRequest)
    - [CreateProjectResponse](#moth-admin-v1-CreateProjectResponse)
    - [DeleteProjectRequest](#moth-admin-v1-DeleteProjectRequest)
    - [DeleteProjectResponse](#moth-admin-v1-DeleteProjectResponse)
    - [EntitlementSpec](#moth-admin-v1-EntitlementSpec)
    - [ExportProjectRequest](#moth-admin-v1-ExportProjectRequest)
    - [ExportProjectResponse](#moth-admin-v1-ExportProjectResponse)
    - [ExportedIdentity](#moth-admin-v1-ExportedIdentity)
    - [ExportedUser](#moth-admin-v1-ExportedUser)
    - [GetProjectRequest](#moth-admin-v1-GetProjectRequest)
    - [GetProjectResponse](#moth-admin-v1-GetProjectResponse)
    - [GetSigningKeyRequest](#moth-admin-v1-GetSigningKeyRequest)
    - [GetSigningKeyResponse](#moth-admin-v1-GetSigningKeyResponse)
    - [GoogleProviderConfig](#moth-admin-v1-GoogleProviderConfig)
    - [ImportProjectRequest](#moth-admin-v1-ImportProjectRequest)
    - [ImportProjectResponse](#moth-admin-v1-ImportProjectResponse)
    - [ImportedUser](#moth-admin-v1-ImportedUser)
    - [ListProjectsRequest](#moth-admin-v1-ListProjectsRequest)
    - [ListProjectsResponse](#moth-admin-v1-ListProjectsResponse)
    - [MonetizationSpec](#moth-admin-v1-MonetizationSpec)
    - [ProductSpec](#moth-admin-v1-ProductSpec)
    - [Project](#moth-admin-v1-Project)
    - [ProjectSettings](#moth-admin-v1-ProjectSettings)
    - [ProjectSpec](#moth-admin-v1-ProjectSpec)
    - [RegenerateSecretKeyRequest](#moth-admin-v1-RegenerateSecretKeyRequest)
    - [RegenerateSecretKeyResponse](#moth-admin-v1-RegenerateSecretKeyResponse)
    - [ResetSigningKeyRequest](#moth-admin-v1-ResetSigningKeyRequest)
    - [ResetSigningKeyResponse](#moth-admin-v1-ResetSigningKeyResponse)
    - [RotateSigningKeyRequest](#moth-admin-v1-RotateSigningKeyRequest)
    - [RotateSigningKeyResponse](#moth-admin-v1-RotateSigningKeyResponse)
    - [SigningKey](#moth-admin-v1-SigningKey)
    - [UpdateProjectRequest](#moth-admin-v1-UpdateProjectRequest)
    - [UpdateProjectResponse](#moth-admin-v1-UpdateProjectResponse)
  
    - [ProjectService](#moth-admin-v1-ProjectService)
  
- [moth/admin/v1/settings.proto](#moth_admin_v1_settings-proto)
    - [GetInstanceSettingsRequest](#moth-admin-v1-GetInstanceSettingsRequest)
    - [GetInstanceSettingsResponse](#moth-admin-v1-GetInstanceSettingsResponse)
    - [SendTestEmailRequest](#moth-admin-v1-SendTestEmailRequest)
    - [SendTestEmailResponse](#moth-admin-v1-SendTestEmailResponse)
    - [SmtpSettings](#moth-admin-v1-SmtpSettings)
    - [UpdateSmtpSettingsRequest](#moth-admin-v1-UpdateSmtpSettingsRequest)
    - [UpdateSmtpSettingsResponse](#moth-admin-v1-UpdateSmtpSettingsResponse)
  
    - [SmtpSource](#moth-admin-v1-SmtpSource)
  
    - [InstanceSettingsService](#moth-admin-v1-InstanceSettingsService)
  
- [moth/admin/v1/user.proto](#moth_admin_v1_user-proto)
    - [CreateUserRequest](#moth-admin-v1-CreateUserRequest)
    - [CreateUserResponse](#moth-admin-v1-CreateUserResponse)
    - [DeleteUserRequest](#moth-admin-v1-DeleteUserRequest)
    - [DeleteUserResponse](#moth-admin-v1-DeleteUserResponse)
    - [DisableUserRequest](#moth-admin-v1-DisableUserRequest)
    - [DisableUserResponse](#moth-admin-v1-DisableUserResponse)
    - [EnableUserRequest](#moth-admin-v1-EnableUserRequest)
    - [EnableUserResponse](#moth-admin-v1-EnableUserResponse)
    - [GetUserRequest](#moth-admin-v1-GetUserRequest)
    - [GetUserResponse](#moth-admin-v1-GetUserResponse)
    - [Identity](#moth-admin-v1-Identity)
    - [ListUsersRequest](#moth-admin-v1-ListUsersRequest)
    - [ListUsersResponse](#moth-admin-v1-ListUsersResponse)
    - [RevokeUserSessionsRequest](#moth-admin-v1-RevokeUserSessionsRequest)
    - [RevokeUserSessionsResponse](#moth-admin-v1-RevokeUserSessionsResponse)
    - [SendPasswordResetRequest](#moth-admin-v1-SendPasswordResetRequest)
    - [SendPasswordResetResponse](#moth-admin-v1-SendPasswordResetResponse)
    - [UpdateUserRequest](#moth-admin-v1-UpdateUserRequest)
    - [UpdateUserResponse](#moth-admin-v1-UpdateUserResponse)
    - [User](#moth-admin-v1-User)
    - [UserSession](#moth-admin-v1-UserSession)
  
    - [UserService](#moth-admin-v1-UserService)
  
- [moth/auth/v1/auth.proto](#moth_auth_v1_auth-proto)
    - [ChangePasswordRequest](#moth-auth-v1-ChangePasswordRequest)
    - [ChangePasswordResponse](#moth-auth-v1-ChangePasswordResponse)
    - [ConfirmEmailChangeRequest](#moth-auth-v1-ConfirmEmailChangeRequest)
    - [ConfirmEmailChangeResponse](#moth-auth-v1-ConfirmEmailChangeResponse)
    - [ConfirmEmailVerificationRequest](#moth-auth-v1-ConfirmEmailVerificationRequest)
    - [ConfirmEmailVerificationResponse](#moth-auth-v1-ConfirmEmailVerificationResponse)
    - [ConfirmPasswordResetRequest](#moth-auth-v1-ConfirmPasswordResetRequest)
    - [ConfirmPasswordResetResponse](#moth-auth-v1-ConfirmPasswordResetResponse)
    - [DeleteAccountRequest](#moth-auth-v1-DeleteAccountRequest)
    - [DeleteAccountResponse](#moth-auth-v1-DeleteAccountResponse)
    - [ExchangeOAuthCodeRequest](#moth-auth-v1-ExchangeOAuthCodeRequest)
    - [ExchangeOAuthCodeResponse](#moth-auth-v1-ExchangeOAuthCodeResponse)
    - [GetMeRequest](#moth-auth-v1-GetMeRequest)
    - [GetMeResponse](#moth-auth-v1-GetMeResponse)
    - [RefreshTokenRequest](#moth-auth-v1-RefreshTokenRequest)
    - [RefreshTokenResponse](#moth-auth-v1-RefreshTokenResponse)
    - [RequestEmailChangeRequest](#moth-auth-v1-RequestEmailChangeRequest)
    - [RequestEmailChangeResponse](#moth-auth-v1-RequestEmailChangeResponse)
    - [RequestEmailVerificationRequest](#moth-auth-v1-RequestEmailVerificationRequest)
    - [RequestEmailVerificationResponse](#moth-auth-v1-RequestEmailVerificationResponse)
    - [RequestPasswordResetRequest](#moth-auth-v1-RequestPasswordResetRequest)
    - [RequestPasswordResetResponse](#moth-auth-v1-RequestPasswordResetResponse)
    - [SignInRequest](#moth-auth-v1-SignInRequest)
    - [SignInResponse](#moth-auth-v1-SignInResponse)
    - [SignInWithOAuthRequest](#moth-auth-v1-SignInWithOAuthRequest)
    - [SignInWithOAuthResponse](#moth-auth-v1-SignInWithOAuthResponse)
    - [SignOutRequest](#moth-auth-v1-SignOutRequest)
    - [SignOutResponse](#moth-auth-v1-SignOutResponse)
    - [SignUpRequest](#moth-auth-v1-SignUpRequest)
    - [SignUpResponse](#moth-auth-v1-SignUpResponse)
    - [TokenPair](#moth-auth-v1-TokenPair)
    - [UnlinkIdentityRequest](#moth-auth-v1-UnlinkIdentityRequest)
    - [UnlinkIdentityResponse](#moth-auth-v1-UnlinkIdentityResponse)
    - [UpdateMeRequest](#moth-auth-v1-UpdateMeRequest)
    - [UpdateMeResponse](#moth-auth-v1-UpdateMeResponse)
    - [User](#moth-auth-v1-User)
  
    - [OAuthProvider](#moth-auth-v1-OAuthProvider)
  
    - [AuthService](#moth-auth-v1-AuthService)
  
- [moth/auth/v1/config.proto](#moth_auth_v1_config-proto)
    - [AppleConfig](#moth-auth-v1-AppleConfig)
    - [Copy](#moth-auth-v1-Copy)
    - [Copy.MessagesEntry](#moth-auth-v1-Copy-MessagesEntry)
    - [GetProjectConfigRequest](#moth-auth-v1-GetProjectConfigRequest)
    - [GetProjectConfigResponse](#moth-auth-v1-GetProjectConfigResponse)
    - [GoogleConfig](#moth-auth-v1-GoogleConfig)
    - [Theme](#moth-auth-v1-Theme)
    - [ThemeColors](#moth-auth-v1-ThemeColors)
  
    - [ConfigService](#moth-auth-v1-ConfigService)
  
- [moth/billing/v1/billing.proto](#moth_billing_v1_billing-proto)
    - [ActiveSubscription](#moth-billing-v1-ActiveSubscription)
    - [Copy](#moth-billing-v1-Copy)
    - [Copy.MessagesEntry](#moth-billing-v1-Copy-MessagesEntry)
    - [CreateBillingPortalSessionRequest](#moth-billing-v1-CreateBillingPortalSessionRequest)
    - [CreateBillingPortalSessionResponse](#moth-billing-v1-CreateBillingPortalSessionResponse)
    - [CreateCheckoutSessionRequest](#moth-billing-v1-CreateCheckoutSessionRequest)
    - [CreateCheckoutSessionResponse](#moth-billing-v1-CreateCheckoutSessionResponse)
    - [CustomerInfo](#moth-billing-v1-CustomerInfo)
    - [Entitlement](#moth-billing-v1-Entitlement)
    - [GetCustomerInfoRequest](#moth-billing-v1-GetCustomerInfoRequest)
    - [GetCustomerInfoResponse](#moth-billing-v1-GetCustomerInfoResponse)
    - [GetOfferingsRequest](#moth-billing-v1-GetOfferingsRequest)
    - [GetOfferingsResponse](#moth-billing-v1-GetOfferingsResponse)
    - [GetPaywallRequest](#moth-billing-v1-GetPaywallRequest)
    - [GetPaywallResponse](#moth-billing-v1-GetPaywallResponse)
    - [Offering](#moth-billing-v1-Offering)
    - [OfferingProduct](#moth-billing-v1-OfferingProduct)
    - [Paywall](#moth-billing-v1-Paywall)
    - [RestorePurchasesRequest](#moth-billing-v1-RestorePurchasesRequest)
    - [RestorePurchasesResponse](#moth-billing-v1-RestorePurchasesResponse)
    - [SubmitPurchaseRequest](#moth-billing-v1-SubmitPurchaseRequest)
    - [SubmitPurchaseResponse](#moth-billing-v1-SubmitPurchaseResponse)
  
    - [EntitlementSource](#moth-billing-v1-EntitlementSource)
    - [PaywallLayout](#moth-billing-v1-PaywallLayout)
    - [Store](#moth-billing-v1-Store)
    - [SubscriptionStatus](#moth-billing-v1-SubscriptionStatus)
  
    - [BillingService](#moth-billing-v1-BillingService)
  
- [moth/projectconfig/v1/projectconfig.proto](#moth_projectconfig_v1_projectconfig-proto)
    - [CacheEnvelope](#moth-projectconfig-v1-CacheEnvelope)
    - [CopyLocaleMessages](#moth-projectconfig-v1-CopyLocaleMessages)
    - [CopyLocaleMessages.MessagesEntry](#moth-projectconfig-v1-CopyLocaleMessages-MessagesEntry)
    - [LegalLinks](#moth-projectconfig-v1-LegalLinks)
    - [StoredCopy](#moth-projectconfig-v1-StoredCopy)
    - [StoredCopy.LocalesEntry](#moth-projectconfig-v1-StoredCopy-LocalesEntry)
    - [StoredPaywall](#moth-projectconfig-v1-StoredPaywall)
    - [StoredTheme](#moth-projectconfig-v1-StoredTheme)
    - [ThemeColorOverrides](#moth-projectconfig-v1-ThemeColorOverrides)
    - [ThemeColors](#moth-projectconfig-v1-ThemeColors)
    - [ThemeLogo](#moth-projectconfig-v1-ThemeLogo)
    - [ThemeShape](#moth-projectconfig-v1-ThemeShape)
    - [ThemeSpacing](#moth-projectconfig-v1-ThemeSpacing)
    - [ThemeTypography](#moth-projectconfig-v1-ThemeTypography)
  
- [moth/server/v1/entitlement.proto](#moth_server_v1_entitlement-proto)
    - [Entitlement](#moth-server-v1-Entitlement)
    - [GetUserEntitlementsRequest](#moth-server-v1-GetUserEntitlementsRequest)
    - [GetUserEntitlementsResponse](#moth-server-v1-GetUserEntitlementsResponse)
  
    - [EntitlementService](#moth-server-v1-EntitlementService)
  
- [moth/server/v1/token.proto](#moth_server_v1_token-proto)
    - [IntrospectTokenRequest](#moth-server-v1-IntrospectTokenRequest)
    - [IntrospectTokenResponse](#moth-server-v1-IntrospectTokenResponse)
  
    - [TokenService](#moth-server-v1-TokenService)
  
- [moth/server/v1/user.proto](#moth_server_v1_user-proto)
    - [CreateUserRequest](#moth-server-v1-CreateUserRequest)
    - [CreateUserResponse](#moth-server-v1-CreateUserResponse)
    - [DeleteUserRequest](#moth-server-v1-DeleteUserRequest)
    - [DeleteUserResponse](#moth-server-v1-DeleteUserResponse)
    - [DisableUserRequest](#moth-server-v1-DisableUserRequest)
    - [DisableUserResponse](#moth-server-v1-DisableUserResponse)
    - [EnableUserRequest](#moth-server-v1-EnableUserRequest)
    - [EnableUserResponse](#moth-server-v1-EnableUserResponse)
    - [GetUserRequest](#moth-server-v1-GetUserRequest)
    - [GetUserResponse](#moth-server-v1-GetUserResponse)
    - [ListUsersRequest](#moth-server-v1-ListUsersRequest)
    - [ListUsersResponse](#moth-server-v1-ListUsersResponse)
    - [RevokeUserSessionsRequest](#moth-server-v1-RevokeUserSessionsRequest)
    - [RevokeUserSessionsResponse](#moth-server-v1-RevokeUserSessionsResponse)
    - [UpdateUserRequest](#moth-server-v1-UpdateUserRequest)
    - [UpdateUserResponse](#moth-server-v1-UpdateUserResponse)
    - [User](#moth-server-v1-User)
  
    - [UserService](#moth-server-v1-UserService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="moth_admin_v1_session-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/session.proto



<a name="moth-admin-v1-Admin"></a>

### Admin



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| email | [string](#string) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="moth-admin-v1-GetCurrentAdminRequest"></a>

### GetCurrentAdminRequest







<a name="moth-admin-v1-GetCurrentAdminResponse"></a>

### GetCurrentAdminResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| admin | [Admin](#moth-admin-v1-Admin) |  |  |
| server_version | [string](#string) |  | The moth build version of the answering server (&#34;dev&#34;, or &#34;vX.Y.Z&#34; on release builds), so a CLI can validate a context and report what it talks to in one round trip. |






<a name="moth-admin-v1-LoginRequest"></a>

### LoginRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |
| password | [string](#string) |  |  |






<a name="moth-admin-v1-LoginResponse"></a>

### LoginResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| admin | [Admin](#moth-admin-v1-Admin) |  |  |






<a name="moth-admin-v1-LogoutRequest"></a>

### LogoutRequest







<a name="moth-admin-v1-LogoutResponse"></a>

### LogoutResponse






 

 

 


<a name="moth-admin-v1-SessionService"></a>

### SessionService
SessionService authenticates operators of the moth instance.

Login sets an HttpOnly session cookie on its HTTP response; every other
admin RPC is authenticated by an interceptor that validates that cookie.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Login | [LoginRequest](#moth-admin-v1-LoginRequest) | [LoginResponse](#moth-admin-v1-LoginResponse) |  |
| Logout | [LogoutRequest](#moth-admin-v1-LogoutRequest) | [LogoutResponse](#moth-admin-v1-LogoutResponse) |  |
| GetCurrentAdmin | [GetCurrentAdminRequest](#moth-admin-v1-GetCurrentAdminRequest) | [GetCurrentAdminResponse](#moth-admin-v1-GetCurrentAdminResponse) |  |

 



<a name="moth_admin_v1_account-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/account.proto



<a name="moth-admin-v1-AcceptAdminInviteRequest"></a>

### AcceptAdminInviteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |
| password | [string](#string) |  |  |






<a name="moth-admin-v1-AcceptAdminInviteResponse"></a>

### AcceptAdminInviteResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| admin | [Admin](#moth-admin-v1-Admin) |  |  |






<a name="moth-admin-v1-AdminInvite"></a>

### AdminInvite
AdminInvite is a pending operator invitation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| email | [string](#string) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| expire_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="moth-admin-v1-ChangePasswordRequest"></a>

### ChangePasswordRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| current_password | [string](#string) |  |  |
| new_password | [string](#string) |  |  |






<a name="moth-admin-v1-ChangePasswordResponse"></a>

### ChangePasswordResponse







<a name="moth-admin-v1-CreatePersonalAccessTokenRequest"></a>

### CreatePersonalAccessTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| expires_in_days | [int32](#int32) |  | Days until the token expires; 0 means it never expires. |






<a name="moth-admin-v1-CreatePersonalAccessTokenResponse"></a>

### CreatePersonalAccessTokenResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  | The `moth_pat_...` plaintext, returned exactly once. |
| metadata | [PersonalAccessToken](#moth-admin-v1-PersonalAccessToken) |  |  |






<a name="moth-admin-v1-InviteAdminRequest"></a>

### InviteAdminRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |






<a name="moth-admin-v1-InviteAdminResponse"></a>

### InviteAdminResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| invite | [AdminInvite](#moth-admin-v1-AdminInvite) |  |  |
| invite_url | [string](#string) |  | Absolute invite URL, returned exactly once. Anyone opening it can claim the account. |
| emailed | [bool](#bool) |  | Whether the invite was also delivered by email. |






<a name="moth-admin-v1-ListAdminInvitesRequest"></a>

### ListAdminInvitesRequest







<a name="moth-admin-v1-ListAdminInvitesResponse"></a>

### ListAdminInvitesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| invites | [AdminInvite](#moth-admin-v1-AdminInvite) | repeated |  |






<a name="moth-admin-v1-ListAdminsRequest"></a>

### ListAdminsRequest







<a name="moth-admin-v1-ListAdminsResponse"></a>

### ListAdminsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| admins | [Admin](#moth-admin-v1-Admin) | repeated |  |






<a name="moth-admin-v1-ListPersonalAccessTokensRequest"></a>

### ListPersonalAccessTokensRequest







<a name="moth-admin-v1-ListPersonalAccessTokensResponse"></a>

### ListPersonalAccessTokensResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tokens | [PersonalAccessToken](#moth-admin-v1-PersonalAccessToken) | repeated | Newest first; revoked tokens are included until they are pruned. |






<a name="moth-admin-v1-PersonalAccessToken"></a>

### PersonalAccessToken
PersonalAccessToken is the metadata of one `moth_pat_...` credential.
The token value itself is stored hashed and only ever returned by
CreatePersonalAccessToken.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| name | [string](#string) |  | Operator-chosen label (&#34;ci&#34;, &#34;laptop&#34;). |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| last_used_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the token last authenticated a request; unset when never used. |
| expire_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Unset means the token never expires. |
| revoke_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Set once the token has been revoked. |






<a name="moth-admin-v1-RevokeAdminInviteRequest"></a>

### RevokeAdminInviteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="moth-admin-v1-RevokeAdminInviteResponse"></a>

### RevokeAdminInviteResponse







<a name="moth-admin-v1-RevokePersonalAccessTokenRequest"></a>

### RevokePersonalAccessTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="moth-admin-v1-RevokePersonalAccessTokenResponse"></a>

### RevokePersonalAccessTokenResponse






 

 

 


<a name="moth-admin-v1-AdminAccountService"></a>

### AdminAccountService
AdminAccountService manages the operator accounts of this moth instance:
inviting additional admins and changing one&#39;s own password. All RPCs
except AcceptAdminInvite require an authenticated admin session.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListAdmins | [ListAdminsRequest](#moth-admin-v1-ListAdminsRequest) | [ListAdminsResponse](#moth-admin-v1-ListAdminsResponse) |  |
| InviteAdmin | [InviteAdminRequest](#moth-admin-v1-InviteAdminRequest) | [InviteAdminResponse](#moth-admin-v1-InviteAdminResponse) | InviteAdmin creates a single-use invite for a new operator account. The invite link is returned for copy-paste and additionally emailed when a mailer is configured. |
| ListAdminInvites | [ListAdminInvitesRequest](#moth-admin-v1-ListAdminInvitesRequest) | [ListAdminInvitesResponse](#moth-admin-v1-ListAdminInvitesResponse) |  |
| RevokeAdminInvite | [RevokeAdminInviteRequest](#moth-admin-v1-RevokeAdminInviteRequest) | [RevokeAdminInviteResponse](#moth-admin-v1-RevokeAdminInviteResponse) |  |
| AcceptAdminInvite | [AcceptAdminInviteRequest](#moth-admin-v1-AcceptAdminInviteRequest) | [AcceptAdminInviteResponse](#moth-admin-v1-AcceptAdminInviteResponse) | AcceptAdminInvite redeems an invite token, creates the admin account and signs it in (sets the session cookie). Unauthenticated. |
| ChangePassword | [ChangePasswordRequest](#moth-admin-v1-ChangePasswordRequest) | [ChangePasswordResponse](#moth-admin-v1-ChangePasswordResponse) | ChangePassword changes the calling admin&#39;s password after verifying the current one, and ends the admin&#39;s other browser sessions. |
| CreatePersonalAccessToken | [CreatePersonalAccessTokenRequest](#moth-admin-v1-CreatePersonalAccessTokenRequest) | [CreatePersonalAccessTokenResponse](#moth-admin-v1-CreatePersonalAccessTokenResponse) | CreatePersonalAccessToken mints a `moth_pat_...` credential for the calling admin, accepted by every admin RPC as `authorization: Bearer` metadata (the moth CLI&#39;s credential). Stored hashed; the plaintext is returned exactly once, in this response. A token minted over a PAT never outlives the creating token: its expiry is capped at the creator&#39;s, so a leaked short-lived token cannot be laundered into a longer-lived one. |
| ListPersonalAccessTokens | [ListPersonalAccessTokensRequest](#moth-admin-v1-ListPersonalAccessTokensRequest) | [ListPersonalAccessTokensResponse](#moth-admin-v1-ListPersonalAccessTokensResponse) | ListPersonalAccessTokens returns the calling admin&#39;s tokens (metadata only, revoked ones included), newest first. |
| RevokePersonalAccessToken | [RevokePersonalAccessTokenRequest](#moth-admin-v1-RevokePersonalAccessTokenRequest) | [RevokePersonalAccessTokenResponse](#moth-admin-v1-RevokePersonalAccessTokenResponse) | RevokePersonalAccessToken immediately ends one of the calling admin&#39;s tokens; its next use fails UNAUTHENTICATED. |

 



<a name="moth_admin_v1_analytics-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/analytics.proto



<a name="moth-admin-v1-CurrencyAmount"></a>

### CurrencyAmount
Money is always reported per currency — never blended. amount_micros is
store-reported gross (net of refunds), not net of store commission or tax.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| currency | [string](#string) |  | ISO-4217 code; empty when the underlying events carried no currency. |
| amount_micros | [int64](#int64) |  |  |






<a name="moth-admin-v1-DailyStat"></a>

### DailyStat
DailyStat is one day of the time series. Days without traffic are
zero-filled so charts render contiguous ranges.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| date | [string](#string) |  | &#34;YYYY-MM-DD&#34; in the project&#39;s rollup timezone. |
| signups | [int64](#int64) |  |  |
| logins | [int64](#int64) |  |  |
| dau | [int64](#int64) |  | Distinct users with a login or token-refresh event that day. |
| failures | [int64](#int64) |  | Failed login attempts (no user attribution). |






<a name="moth-admin-v1-Event"></a>

### Event
Event is one server-emitted analytics event, trimmed for the activity
feed. PII-minimal by construction: the user id only — no email, no
metadata payload, no IP, no device ids.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| type | [string](#string) |  | Canonical event type, e.g. &#34;user.signup&#34;, &#34;user.login&#34;, &#34;user.login_failed&#34;, &#34;token.refresh&#34;, &#34;user.deleted&#34;. |
| user_id | [string](#string) |  | Empty for events without a subject (login failures). |
| provider | [string](#string) |  | Identity provider involved (&#34;password&#34;, &#34;google&#34;, &#34;apple&#34;); empty when not applicable. |
| platform | [string](#string) |  | SDK-reported platform (&#34;ios&#34;, &#34;android&#34;, &#34;web&#34;); empty when none was reported. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="moth-admin-v1-GetStatsRequest"></a>

### GetStatsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| from_date | [string](#string) |  | First day of the range, &#34;YYYY-MM-DD&#34; in the project&#39;s rollup timezone. |
| to_date | [string](#string) |  | Last day of the range (inclusive), same format. |
| granularity | [Granularity](#moth-admin-v1-Granularity) |  | Bucket size; unspecified means DAY. |






<a name="moth-admin-v1-GetStatsResponse"></a>

### GetStatsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tiles | [StatTiles](#moth-admin-v1-StatTiles) |  |  |
| series | [DailyStat](#moth-admin-v1-DailyStat) | repeated | One entry per day in [from_date, to_date], oldest first, zero-filled. |
| providers | [ProviderBreakdown](#moth-admin-v1-ProviderBreakdown) |  | Totals over the requested range. |
| platforms | [PlatformBreakdown](#moth-admin-v1-PlatformBreakdown) |  |  |






<a name="moth-admin-v1-GetSubscriptionStatsRequest"></a>

### GetSubscriptionStatsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| from_period | [string](#string) |  | First month of the range, &#34;YYYY-MM&#34; in the project&#39;s rollup timezone. |
| to_period | [string](#string) |  | Last month of the range (inclusive), same format. |






<a name="moth-admin-v1-GetSubscriptionStatsResponse"></a>

### GetSubscriptionStatsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tiles | [SubscriptionTiles](#moth-admin-v1-SubscriptionTiles) |  |  |
| series | [SubscriptionMonthlyStat](#moth-admin-v1-SubscriptionMonthlyStat) | repeated | One entry per month in [from_period, to_period], oldest first, zero-filled. |
| tiers | [SubscriptionTierBreakdown](#moth-admin-v1-SubscriptionTierBreakdown) | repeated | Per-tier revenue/subscriber breakdown over the requested range. |
| stores | [SubscriptionStoreBreakdown](#moth-admin-v1-SubscriptionStoreBreakdown) |  | Per-store revenue breakdown over the requested range. |






<a name="moth-admin-v1-ListRecentEventsRequest"></a>

### ListRecentEventsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| limit | [int32](#int32) |  | Maximum events to return; the server defaults and caps it (50). |






<a name="moth-admin-v1-ListRecentEventsResponse"></a>

### ListRecentEventsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| events | [Event](#moth-admin-v1-Event) | repeated | Newest first. |






<a name="moth-admin-v1-PlatformBreakdown"></a>

### PlatformBreakdown
PlatformBreakdown splits the range&#39;s logins by SDK-reported platform;
other collects everything that is not ios/android/web, including logins
with no reported platform.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| ios | [int64](#int64) |  |  |
| android | [int64](#int64) |  |  |
| web | [int64](#int64) |  |  |
| other | [int64](#int64) |  |  |






<a name="moth-admin-v1-ProviderBreakdown"></a>

### ProviderBreakdown
ProviderBreakdown splits the range&#39;s logins by identity provider.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| password | [int64](#int64) |  |  |
| google | [int64](#int64) |  |  |
| apple | [int64](#int64) |  |  |






<a name="moth-admin-v1-RunRollupRequest"></a>

### RunRollupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  | Project to roll up; empty rolls up every project. |






<a name="moth-admin-v1-RunRollupResponse"></a>

### RunRollupResponse
RunRollupResponse summarizes the completed run (also recorded server-side
for observability).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| run_id | [string](#string) |  |  |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| finish_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| days_processed | [int32](#int32) |  | Number of (project, day) windows re-aggregated. |
| events_pruned | [int64](#int64) |  | Raw events removed by retention pruning. |






<a name="moth-admin-v1-StatTiles"></a>

### StatTiles
StatTiles is the headline block of the project analytics tab. The 7-day
figures cover the last 7 rolled-up days.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| total_users | [int64](#int64) |  | End-user accounts in the project right now (all time, live count). |
| new_users_7d | [int64](#int64) |  | Signups over the last 7 days, and over the 7 days before those (the pair renders the trend arrow). |
| new_users_previous_7d | [int64](#int64) |  |  |
| latest_dau | [int64](#int64) |  | Distinct active users on the most recent rolled-up day (usually yesterday) and which day that was (&#34;YYYY-MM-DD&#34;, empty when no data). DAU approximates &#34;active&#34; as: had a login or token-refresh event. |
| latest_dau_date | [string](#string) |  |  |
| logins_7d | [int64](#int64) |  | Login attempts over the last 7 days, split by outcome. An elevated failure count is an ops signal (misconfigured provider, expired key). |
| login_failures_7d | [int64](#int64) |  |  |
| login_success_rate_7d | [double](#double) |  | logins / (logins &#43; failures) over the last 7 days, 0..1. Zero when there were no attempts — check the raw counts before rendering. |






<a name="moth-admin-v1-SubscriptionMonthlyStat"></a>

### SubscriptionMonthlyStat
SubscriptionMonthlyStat is one month of the time series. Revenue and the
per-store split are per currency; new/renewed/churned/trial counts are event
counts summed across currencies, while active_subscribers is a currency-
agnostic distinct-user count (not a per-currency sum). Months without data
are zero-filled so charts render contiguous ranges.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| period | [string](#string) |  | &#34;YYYY-MM&#34; in the project&#39;s rollup timezone. |
| revenue | [CurrencyAmount](#moth-admin-v1-CurrencyAmount) | repeated | Total revenue this month, per currency (net of refunds). |
| active_subscribers | [int64](#int64) |  |  |
| new_subscribers | [int64](#int64) |  |  |
| renewals | [int64](#int64) |  |  |
| churned | [int64](#int64) |  |  |
| trials_started | [int64](#int64) |  |  |
| trials_converted | [int64](#int64) |  |  |






<a name="moth-admin-v1-SubscriptionStoreBreakdown"></a>

### SubscriptionStoreBreakdown
SubscriptionStoreBreakdown splits the range&#39;s revenue by store, per currency.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| apple | [CurrencyAmount](#moth-admin-v1-CurrencyAmount) | repeated |  |
| google | [CurrencyAmount](#moth-admin-v1-CurrencyAmount) | repeated |  |
| stripe | [CurrencyAmount](#moth-admin-v1-CurrencyAmount) | repeated |  |






<a name="moth-admin-v1-SubscriptionTierBreakdown"></a>

### SubscriptionTierBreakdown
SubscriptionTierBreakdown is one product&#39;s share of the requested range.
Revenue (per currency) and new_subscribers sum over the whole range; because
active_subscribers is a non-additive distinct-user count, it is that
product&#39;s currency-agnostic distinct count for the latest month in the range
(matching the headline tile), not a sum across months.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| product_id | [string](#string) |  | Empty for events with no mapped moth product. |
| revenue | [CurrencyAmount](#moth-admin-v1-CurrencyAmount) | repeated |  |
| new_subscribers | [int64](#int64) |  |  |
| active_subscribers | [int64](#int64) |  |  |






<a name="moth-admin-v1-SubscriptionTiles"></a>

### SubscriptionTiles
SubscriptionTiles is the headline block of the subscriptions section — the
current (latest rolled-up) month against the previous one.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| latest_period | [string](#string) |  | The latest rolled-up month (&#34;YYYY-MM&#34;, empty when no data) these figures describe, and the month before it for the trend arrows. |
| revenue_this_month | [CurrencyAmount](#moth-admin-v1-CurrencyAmount) | repeated | Total revenue this month and last month, per currency (the headline). |
| revenue_previous_month | [CurrencyAmount](#moth-admin-v1-CurrencyAmount) | repeated |  |
| active_subscribers | [int64](#int64) |  | Active subscribers this month and last: distinct users with an active event (purchased, renewed or trial_started), counted across all currencies as a single distinct-user total — never the sum of per-currency counts, which would double-count a user transacting in more than one currency. |
| active_subscribers_previous | [int64](#int64) |  |  |
| new_subscribers | [int64](#int64) |  |  |
| churned | [int64](#int64) |  |  |
| trials_started | [int64](#int64) |  |  |
| trials_converted | [int64](#int64) |  |  |
| trial_conversion_rate | [double](#double) |  | trials_converted / trials_started over the latest month, 0..1. Zero when no trials started — check the raw counts before rendering. |





 


<a name="moth-admin-v1-Granularity"></a>

### Granularity
Granularity selects the time-series bucket size. Only daily buckets are
rolled up today.

| Name | Number | Description |
| ---- | ------ | ----------- |
| GRANULARITY_UNSPECIFIED | 0 |  |
| GRANULARITY_DAY | 1 |  |


 

 


<a name="moth-admin-v1-AnalyticsService"></a>

### AnalyticsService
AnalyticsService serves the project analytics dashboards. Numbers come
from pre-aggregated daily rollups (daily_stats) so dashboards never scan
the raw event stream; the activity feed reads the newest raw events.
Dates are calendar days (&#34;YYYY-MM-DD&#34;) bucketed in the project&#39;s rollup
timezone (ProjectSettings.rollup_timezone). All RPCs require an
authenticated admin session.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetStats | [GetStatsRequest](#moth-admin-v1-GetStatsRequest) | [GetStatsResponse](#moth-admin-v1-GetStatsResponse) | GetStats returns the stat tiles, the per-day time series and the provider/platform breakdowns for one project over [from_date, to_date] (inclusive). |
| ListRecentEvents | [ListRecentEventsRequest](#moth-admin-v1-ListRecentEventsRequest) | [ListRecentEventsResponse](#moth-admin-v1-ListRecentEventsResponse) | ListRecentEvents returns the project&#39;s newest raw events for the activity feed, newest first. |
| RunRollup | [RunRollupRequest](#moth-admin-v1-RunRollupRequest) | [RunRollupResponse](#moth-admin-v1-RunRollupResponse) | RunRollup triggers the aggregate-and-prune job immediately — for one project or the whole instance — and returns the run summary. The same job also runs nightly; re-rolling a day is idempotent. |
| GetSubscriptionStats | [GetSubscriptionStatsRequest](#moth-admin-v1-GetSubscriptionStatsRequest) | [GetSubscriptionStatsResponse](#moth-admin-v1-GetSubscriptionStatsResponse) | GetSubscriptionStats returns the milestone-14 subscription revenue tiles, the per-month time series and the per-tier / per-store breakdowns for one project over [from_period, to_period] (inclusive, &#34;YYYY-MM&#34;). Like GetStats, every number comes from the pre-aggregated monthly rollup (subscription_monthly_stats / subscription_tier_stats) — this RPC never scans the raw subscription_events stream. Money is store-reported gross, reported per currency and never blended across currencies (no FX). |

 



<a name="moth_admin_v1_audit-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/audit.proto



<a name="moth-admin-v1-AuditEntry"></a>

### AuditEntry
AuditEntry is one immutable audit record.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| actor_type | [string](#string) |  | Credential kind that performed the action: &#34;cookie&#34; (browser session), &#34;pat&#34; (personal access token) or &#34;system&#34; (the server itself). |
| actor_id | [string](#string) |  | Identifier of the actor (admin id, or empty for system events). |
| actor_label | [string](#string) |  | Human-readable actor label (admin email, PAT name). |
| action | [string](#string) |  | Machine action name, e.g. &#34;project.create&#34;, &#34;user.disable&#34;, &#34;signing_key.rotate&#34;, &#34;provider.update&#34;. |
| target_type | [string](#string) |  | Kind of the affected object (&#34;project&#34;, &#34;user&#34;, &#34;signing_key&#34;, ...). |
| target_id | [string](#string) |  | Identifier of the affected object. |
| project_id | [string](#string) |  | Owning project; empty for instance-level actions. |
| summary | [string](#string) |  | Short human-readable description of what happened. |
| before_after | [string](#string) |  | Optional JSON object describing the before/after of the change. |
| ip | [string](#string) |  | Coarse or hashed client IP. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="moth-admin-v1-ListAuditLogRequest"></a>

### ListAuditLogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  | Optional: only entries of this project. |
| actor_id | [string](#string) |  | Optional: only entries by this actor id. |
| action | [string](#string) |  | Optional: only entries with this exact action name. |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Optional inclusive lower bound on create_time. |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Optional exclusive upper bound on create_time. |
| page_size | [int32](#int32) |  | Page size, 1–200; 0 means the default (50). |
| page_token | [string](#string) |  | next_page_token of the previous response; empty for the first page. |






<a name="moth-admin-v1-ListAuditLogResponse"></a>

### ListAuditLogResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entries | [AuditEntry](#moth-admin-v1-AuditEntry) | repeated |  |
| next_page_token | [string](#string) |  | Empty when this is the last page. |





 

 

 


<a name="moth-admin-v1-AuditService"></a>

### AuditService
AuditService exposes the append-only audit log to the admin console. Every
admin action (through a browser session or a personal access token) and
security-relevant server event is recorded and readable here. All RPCs
require an authenticated admin session.

CSV export is served as a plain-HTTP endpoint (GET /admin/audit.csv,
added by the milestone-10 build stage), not as an RPC, so the browser can
stream and download it directly.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListAuditLog | [ListAuditLogRequest](#moth-admin-v1-ListAuditLogRequest) | [ListAuditLogResponse](#moth-admin-v1-ListAuditLogResponse) | ListAuditLog returns audit entries newest-first, narrowed by the optional filters and paged with page_size / page_token. |

 



<a name="moth_admin_v1_billing_credentials-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/billing_credentials.proto



<a name="moth-admin-v1-AppleBillingConfig"></a>

### AppleBillingConfig
AppleBillingConfig is a project&#39;s App Store Server API configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| iap_key_id | [string](#string) |  | In-App-Purchase key id (from App Store Connect). |
| iap_issuer_id | [string](#string) |  | Issuer id of the In-App-Purchase key. |
| iap_key_p8 | [string](#string) |  | Contents of the `.p8` In-App-Purchase private key. Write-only: accepted on update, never returned; empty keeps the stored one. Stored encrypted. |
| has_iap_key | [bool](#bool) |  | Output only: whether a `.p8` key is stored. |
| bundle_id | [string](#string) |  | App bundle id (the `aud`/`bid` of signed transactions). |
| app_apple_id | [string](#string) |  | The app&#39;s Apple id (numeric App Store id). |
| notification_secret | [string](#string) |  | Shared secret authenticating the App Store Server Notifications webhook. Write-only; empty keeps the stored one. Stored encrypted. |
| has_notification_secret | [bool](#bool) |  | Output only: whether a notification secret is stored. |
| notification_url | [string](#string) |  | The App Store Server Notification URL moth has registered for the app. Apple exposes no read for it, so moth persists what it registered to keep `moth setup billing` idempotent (re-register only on a change). Set by the CLI after a successful registration; empty on update keeps the stored one. |






<a name="moth-admin-v1-GetBillingCredentialsRequest"></a>

### GetBillingCredentialsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-GetBillingCredentialsResponse"></a>

### GetBillingCredentialsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| apple | [AppleBillingConfig](#moth-admin-v1-AppleBillingConfig) |  |  |
| google | [GoogleBillingConfig](#moth-admin-v1-GoogleBillingConfig) |  |  |
| stripe | [StripeBillingConfig](#moth-admin-v1-StripeBillingConfig) |  |  |






<a name="moth-admin-v1-GoogleBillingConfig"></a>

### GoogleBillingConfig
GoogleBillingConfig is a project&#39;s Google Play Developer API configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| service_account_json | [string](#string) |  | Service-account JSON with Play Developer API access. Write-only: accepted on update, never returned; empty keeps the stored one. Stored encrypted. |
| has_service_account | [bool](#bool) |  | Output only: whether a service account is stored. |
| package_name | [string](#string) |  | App package name. |
| pubsub_topic | [string](#string) |  | Cloud Pub/Sub topic RTDN messages are published to. |
| rtdn_secret | [string](#string) |  | Shared secret authenticating the RTDN Pub/Sub push webhook. Write-only; empty keeps the stored one. Stored encrypted. |
| has_rtdn_secret | [bool](#bool) |  | Output only: whether an RTDN secret is stored. |






<a name="moth-admin-v1-StripeBillingConfig"></a>

### StripeBillingConfig
StripeBillingConfig is a project&#39;s Stripe API configuration (milestone 17).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret_key | [string](#string) |  | Stripe secret key (a restricted key is recommended). Write-only: accepted on update, never returned; empty keeps the stored one. Stored encrypted. |
| has_secret_key | [bool](#bool) |  | Output only: whether a secret key is stored. |
| webhook_secret | [string](#string) |  | Webhook signing secret (&#34;whsec_...&#34;) authenticating the Stripe-Signature header. Write-only; empty keeps the stored one. Stored encrypted. |
| has_webhook_secret | [bool](#bool) |  | Output only: whether a webhook secret is stored. |
| webhook_endpoint_id | [string](#string) |  | The Stripe webhook endpoint (&#34;we_...&#34;) moth created via the API, persisted to keep `moth setup billing` idempotent (mirroring the Apple notification_url). Set by the CLI after a successful registration; empty on update keeps the stored one. |






<a name="moth-admin-v1-UpdateBillingCredentialsRequest"></a>

### UpdateBillingCredentialsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| apple | [AppleBillingConfig](#moth-admin-v1-AppleBillingConfig) |  |  |
| google | [GoogleBillingConfig](#moth-admin-v1-GoogleBillingConfig) |  |  |
| stripe | [StripeBillingConfig](#moth-admin-v1-StripeBillingConfig) |  |  |






<a name="moth-admin-v1-UpdateBillingCredentialsResponse"></a>

### UpdateBillingCredentialsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| apple | [AppleBillingConfig](#moth-admin-v1-AppleBillingConfig) |  |  |
| google | [GoogleBillingConfig](#moth-admin-v1-GoogleBillingConfig) |  |  |
| stripe | [StripeBillingConfig](#moth-admin-v1-StripeBillingConfig) |  |  |





 

 

 


<a name="moth-admin-v1-BillingCredentialsService"></a>

### BillingCredentialsService
BillingCredentialsService edits a project&#39;s store API credentials — the
Apple App Store Server API In-App-Purchase key and the Google Play Developer
API service account, plus the webhook shared secrets. Secrets are write-only
(accepted on update, never returned; reads report presence via has_*
indicators) and stored encrypted at rest, like every other moth secret. All
RPCs require an authenticated admin session.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetBillingCredentials | [GetBillingCredentialsRequest](#moth-admin-v1-GetBillingCredentialsRequest) | [GetBillingCredentialsResponse](#moth-admin-v1-GetBillingCredentialsResponse) | GetBillingCredentials returns the non-secret fields and has_* presence indicators for the stored secrets. |
| UpdateBillingCredentials | [UpdateBillingCredentialsRequest](#moth-admin-v1-UpdateBillingCredentialsRequest) | [UpdateBillingCredentialsResponse](#moth-admin-v1-UpdateBillingCredentialsResponse) | UpdateBillingCredentials writes the credentials. For each secret field, an empty value keeps the stored one; reads never echo secrets back. |

 



<a name="moth_admin_v1_copy-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/copy.proto



<a name="moth-admin-v1-CopyKey"></a>

### CopyKey
CopyKey is one catalog entry as the editor needs it: the key, its bundled
default for the requested locale, its override value (empty when the project
has none), and the validation contract (required placeholders &#43; length cap)
the editor should enforce client-side before saving.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  | Stable dotted catalog key, e.g. &#34;sign_in.title&#34;. |
| screen | [CopyScreen](#moth-admin-v1-CopyScreen) |  |  |
| default_value | [string](#string) |  | The bundled default string for the requested locale (the hint shown under an empty override, and what a reset restores). |
| override_value | [string](#string) |  | The project&#39;s current override for this (locale, key); empty when none — the screen renders default_value. |
| placeholders | [string](#string) | repeated | Interpolation placeholders that must appear in any override of this key (e.g. &#34;app_name&#34;, &#34;expiry&#34;); a save missing one is rejected. |
| max_length | [int32](#int32) |  | Maximum length an override of this key may have; 0 means no explicit cap. |






<a name="moth-admin-v1-CopyRevision"></a>

### CopyRevision
CopyRevision is one saved version of a project&#39;s copy overrides.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revision_id | [string](#string) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| locales | [string](#string) | repeated | The locales this revision carried overrides for (for a compact history label); the full document is restored by RestoreCopyRevision. |






<a name="moth-admin-v1-GetProjectCopyRequest"></a>

### GetProjectCopyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| locale | [string](#string) |  | BCP-47 tag to edit (e.g. &#34;fr&#34;). Empty defaults to the project&#39;s default locale. |
| screen | [CopyScreen](#moth-admin-v1-CopyScreen) |  | Restrict the returned keys to one screen; UNSPECIFIED returns every screen&#39;s keys. |






<a name="moth-admin-v1-GetProjectCopyResponse"></a>

### GetProjectCopyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| keys | [CopyKey](#moth-admin-v1-CopyKey) | repeated | Every catalog key for the requested screen(s), with the bundled default for the locale and the project&#39;s current override. |
| locale | [string](#string) |  | The locale these keys resolve for (the request&#39;s locale, or the project default when the request left it empty). |
| revision_id | [string](#string) |  | The project&#39;s current copy revision; empty when it renders the fully bundled default. |
| is_default | [bool](#bool) |  | True when the project has no copy overrides at all. |






<a name="moth-admin-v1-ListCopyRevisionsRequest"></a>

### ListCopyRevisionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| limit | [int32](#int32) |  | Maximum revisions to return; 0 (or anything above what the server keeps) returns all kept revisions. |






<a name="moth-admin-v1-ListCopyRevisionsResponse"></a>

### ListCopyRevisionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revisions | [CopyRevision](#moth-admin-v1-CopyRevision) | repeated | Newest first. |






<a name="moth-admin-v1-ListLocalesRequest"></a>

### ListLocalesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-ListLocalesResponse"></a>

### ListLocalesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| locales | [Locale](#moth-admin-v1-Locale) | repeated | The union of moth&#39;s bundled locales and the project&#39;s customized locales, with the project default flagged. |
| default_locale | [string](#string) |  | The project&#39;s default locale tag. |






<a name="moth-admin-v1-Locale"></a>

### Locale
Locale describes one language the editor can select.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tag | [string](#string) |  | BCP-47 tag, e.g. &#34;fr&#34; or &#34;pt-BR&#34;. |
| display_name | [string](#string) |  | Human-readable name for the selector, e.g. &#34;French&#34;. |
| bundled | [bool](#bool) |  | True when moth ships a bundled translation for this locale. |
| customized | [bool](#bool) |  | True when the project has overrides for this locale. |
| is_default | [bool](#bool) |  | True when this is the project&#39;s default locale. |






<a name="moth-admin-v1-ResetCopyRequest"></a>

### ResetCopyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| locale | [string](#string) |  | BCP-47 tag to reset; required. |
| key | [string](#string) |  | The single key to reset; empty resets the whole locale to bundled defaults. |






<a name="moth-admin-v1-ResetCopyResponse"></a>

### ResetCopyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revision_id | [string](#string) |  | The id of the new revision the reset created; empty when the reset left the project with no overrides (reverted to the fully bundled default). |






<a name="moth-admin-v1-RestoreCopyRevisionRequest"></a>

### RestoreCopyRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| revision_id | [string](#string) |  |  |






<a name="moth-admin-v1-RestoreCopyRevisionResponse"></a>

### RestoreCopyRevisionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revision_id | [string](#string) |  | The id of the new revision created by the restore; empty when the restored document had no overrides. |






<a name="moth-admin-v1-UpdateProjectCopyRequest"></a>

### UpdateProjectCopyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| locale | [string](#string) |  | BCP-47 tag to write; required. |
| values | [UpdateProjectCopyRequest.ValuesEntry](#moth-admin-v1-UpdateProjectCopyRequest-ValuesEntry) | repeated | Full replacement of this locale&#39;s overrides: key → override string. Keys must exist in the catalog and each value is placeholder/length validated; a key omitted here (or set empty) reverts to the bundled default. Clearing every value reverts the whole locale to defaults. |






<a name="moth-admin-v1-UpdateProjectCopyRequest-ValuesEntry"></a>

### UpdateProjectCopyRequest.ValuesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="moth-admin-v1-UpdateProjectCopyResponse"></a>

### UpdateProjectCopyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revision_id | [string](#string) |  | The id of the revision this save created; empty when the update left the project with no overrides (reverted to the fully bundled default). |





 


<a name="moth-admin-v1-CopyScreen"></a>

### CopyScreen
CopyScreen is one SDK/hosted-page surface the catalog groups keys under.
COPY_SCREEN_UNSPECIFIED selects every screen (used to request the whole
editable surface at once).

| Name | Number | Description |
| ---- | ------ | ----------- |
| COPY_SCREEN_UNSPECIFIED | 0 |  |
| COPY_SCREEN_SIGN_IN | 1 |  |
| COPY_SCREEN_SIGN_UP | 2 |  |
| COPY_SCREEN_PASSWORD_RESET | 3 |  |
| COPY_SCREEN_VERIFY_EMAIL | 4 |  |
| COPY_SCREEN_PAYWALL | 5 |  |


 

 


<a name="moth-admin-v1-CopyService"></a>

### CopyService
CopyService manages a project&#39;s localization copy: the per-locale overrides
on top of moth&#39;s bundled message catalog (internal/i18n). The catalog is a
small, closed, curated key set per screen — sign_in.*, sign_up.*,
password_reset.*, verify_email.*, paywall.* — with a bundled default in
every bundled locale, so a project that customizes nothing is still fully
localized. Overrides are validated server-side (key must exist, required
placeholders present, length bounded) so any accepted copy still renders a
sensible screen. Every save creates a revision (the last 10 are kept for
undo). All RPCs require an authenticated admin session. Mirrors
ThemeService (design tokens) — copy keyed by screen × locale instead of by
color.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetProjectCopy | [GetProjectCopyRequest](#moth-admin-v1-GetProjectCopyRequest) | [GetProjectCopyResponse](#moth-admin-v1-GetProjectCopyResponse) | GetProjectCopy returns the editable copy for one locale: every catalog key (optionally filtered to one screen) with its bundled default for that locale plus the project&#39;s current override, so the editor can render the whole editable surface and a reset-to-default hint per key. |
| UpdateProjectCopy | [UpdateProjectCopyRequest](#moth-admin-v1-UpdateProjectCopyRequest) | [UpdateProjectCopyResponse](#moth-admin-v1-UpdateProjectCopyResponse) | UpdateProjectCopy replaces one locale&#39;s overrides with the supplied key → value map (validated against the catalog) and returns the new revision. Keys omitted (or set empty) fall back to the bundled default. |
| ListCopyRevisions | [ListCopyRevisionsRequest](#moth-admin-v1-ListCopyRevisionsRequest) | [ListCopyRevisionsResponse](#moth-admin-v1-ListCopyRevisionsResponse) | ListCopyRevisions returns the saved revisions, newest first (at most the 10 the server keeps). |
| RestoreCopyRevision | [RestoreCopyRevisionRequest](#moth-admin-v1-RestoreCopyRevisionRequest) | [RestoreCopyRevisionResponse](#moth-admin-v1-RestoreCopyRevisionResponse) | RestoreCopyRevision re-installs an old revision&#39;s whole override document as a new revision (history only ever moves forward). |
| ResetCopy | [ResetCopyRequest](#moth-admin-v1-ResetCopyRequest) | [ResetCopyResponse](#moth-admin-v1-ResetCopyResponse) | ResetCopy reverts one key (key set) or a whole locale (key empty) to the bundled default by removing the override(s), as a new revision. Removing the last override reverts the project to the fully bundled default. |
| ListLocales | [ListLocalesRequest](#moth-admin-v1-ListLocalesRequest) | [ListLocalesResponse](#moth-admin-v1-ListLocalesResponse) | ListLocales returns the project&#39;s available locales (its default plus any it has customized) and moth&#39;s bundled locales, so the editor can offer a language selector with an &#34;add language&#34; action. |

 



<a name="moth_admin_v1_entitlement-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/entitlement.proto



<a name="moth-admin-v1-CreateEntitlementRequest"></a>

### CreateEntitlementRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| identifier | [string](#string) |  |  |
| display_name | [string](#string) |  |  |






<a name="moth-admin-v1-CreateEntitlementResponse"></a>

### CreateEntitlementResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entitlement | [Entitlement](#moth-admin-v1-Entitlement) |  |  |






<a name="moth-admin-v1-DeleteEntitlementRequest"></a>

### DeleteEntitlementRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| id | [string](#string) |  |  |






<a name="moth-admin-v1-DeleteEntitlementResponse"></a>

### DeleteEntitlementResponse







<a name="moth-admin-v1-Entitlement"></a>

### Entitlement
Entitlement is one named capability of a project.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| identifier | [string](#string) |  | Stable identifier the app checks (e.g. &#34;pro&#34;); unique within the project. |
| display_name | [string](#string) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="moth-admin-v1-ListEntitlementsRequest"></a>

### ListEntitlementsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-ListEntitlementsResponse"></a>

### ListEntitlementsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entitlements | [Entitlement](#moth-admin-v1-Entitlement) | repeated |  |






<a name="moth-admin-v1-UpdateEntitlementRequest"></a>

### UpdateEntitlementRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| id | [string](#string) |  |  |
| display_name | [string](#string) |  | The identifier is immutable (apps depend on it); only display_name is editable. |






<a name="moth-admin-v1-UpdateEntitlementResponse"></a>

### UpdateEntitlementResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entitlement | [Entitlement](#moth-admin-v1-Entitlement) |  |  |





 

 

 


<a name="moth-admin-v1-EntitlementService"></a>

### EntitlementService
EntitlementService manages a project&#39;s entitlement definitions — the named
capabilities (`pro`, `premium`, …) apps gate features on, decoupled from the
products that grant them. All RPCs require an authenticated admin session.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListEntitlements | [ListEntitlementsRequest](#moth-admin-v1-ListEntitlementsRequest) | [ListEntitlementsResponse](#moth-admin-v1-ListEntitlementsResponse) |  |
| CreateEntitlement | [CreateEntitlementRequest](#moth-admin-v1-CreateEntitlementRequest) | [CreateEntitlementResponse](#moth-admin-v1-CreateEntitlementResponse) |  |
| UpdateEntitlement | [UpdateEntitlementRequest](#moth-admin-v1-UpdateEntitlementRequest) | [UpdateEntitlementResponse](#moth-admin-v1-UpdateEntitlementResponse) |  |
| DeleteEntitlement | [DeleteEntitlementRequest](#moth-admin-v1-DeleteEntitlementRequest) | [DeleteEntitlementResponse](#moth-admin-v1-DeleteEntitlementResponse) | DeleteEntitlement removes the definition; it cascades to product grants and operator grants referencing it. |

 



<a name="moth_admin_v1_product-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/product.proto



<a name="moth-admin-v1-CreateProductRequest"></a>

### CreateProductRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| product | [Product](#moth-admin-v1-Product) |  | The product to create; id and timestamps are ignored (server-assigned). |






<a name="moth-admin-v1-CreateProductResponse"></a>

### CreateProductResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| product | [Product](#moth-admin-v1-Product) |  |  |






<a name="moth-admin-v1-DeleteProductRequest"></a>

### DeleteProductRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| id | [string](#string) |  |  |






<a name="moth-admin-v1-DeleteProductResponse"></a>

### DeleteProductResponse







<a name="moth-admin-v1-GetProductRequest"></a>

### GetProductRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| id | [string](#string) |  |  |






<a name="moth-admin-v1-GetProductResponse"></a>

### GetProductResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| product | [Product](#moth-admin-v1-Product) |  |  |






<a name="moth-admin-v1-ListProductsRequest"></a>

### ListProductsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-ListProductsResponse"></a>

### ListProductsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| products | [Product](#moth-admin-v1-Product) | repeated |  |






<a name="moth-admin-v1-Product"></a>

### Product
Product is one subscription tier. An &#34;offering&#34; is not a separate object: the
products sharing an `offering` tag, ordered by `sort_order`, form one paywall
listing.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| identifier | [string](#string) |  | Internal catalog identifier, unique within the project (e.g. &#34;monthly&#34;). |
| display_name | [string](#string) |  |  |
| apple_product_id | [string](#string) |  | Store SKUs; either may be empty when the tier ships on one store only. |
| google_product_id | [string](#string) |  |  |
| billing_period | [string](#string) |  | Billing period (&#34;weekly&#34;, &#34;monthly&#34;, &#34;yearly&#34;, …); free-form, the store read is authoritative. |
| price_amount_micros | [int64](#int64) |  | List price metadata (display &#43; analytics only). Micros = price * 1_000_000. |
| currency | [string](#string) |  |  |
| trial_period | [string](#string) |  | Intro/trial descriptor (display &#43; analytics only). |
| intro_price_amount_micros | [int64](#int64) |  |  |
| intro_period | [string](#string) |  |  |
| offering | [string](#string) |  | Paywall grouping tag and order within it. |
| sort_order | [int32](#int32) |  |  |
| entitlement_ids | [string](#string) | repeated | The entitlements this product grants while active (entitlement ids). |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| stripe_price_id | [string](#string) |  | Stripe recurring Price id (&#34;price_...&#34;); empty when the tier does not sell on the web. Provisioning (&#34;create in Stripe&#34;) writes it back; a price edit creates a new Price and re-points this field (Stripe prices are immutable). |
| stripe_product_id | [string](#string) |  | The linked Stripe Product id (&#34;prod_...&#34;), written back by provisioning. |






<a name="moth-admin-v1-UpdateProductRequest"></a>

### UpdateProductRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| id | [string](#string) |  |  |
| product | [Product](#moth-admin-v1-Product) |  | The desired product state; entitlement_ids replaces the whole grant set. |






<a name="moth-admin-v1-UpdateProductResponse"></a>

### UpdateProductResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| product | [Product](#moth-admin-v1-Product) |  |  |





 

 

 


<a name="moth-admin-v1-ProductService"></a>

### ProductService
ProductService manages a project&#39;s subscription tiers (products): their store
SKUs, the entitlements they grant, price metadata, and paywall placement.
This is the catalog the store-catalog automation (milestone 12) and the
paywall (milestone 13) consume. All RPCs require an authenticated admin
session.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListProducts | [ListProductsRequest](#moth-admin-v1-ListProductsRequest) | [ListProductsResponse](#moth-admin-v1-ListProductsResponse) | ListProducts returns the project&#39;s products in paywall order (by sort_order). |
| GetProduct | [GetProductRequest](#moth-admin-v1-GetProductRequest) | [GetProductResponse](#moth-admin-v1-GetProductResponse) |  |
| CreateProduct | [CreateProductRequest](#moth-admin-v1-CreateProductRequest) | [CreateProductResponse](#moth-admin-v1-CreateProductResponse) |  |
| UpdateProduct | [UpdateProductRequest](#moth-admin-v1-UpdateProductRequest) | [UpdateProductResponse](#moth-admin-v1-UpdateProductResponse) |  |
| DeleteProduct | [DeleteProductRequest](#moth-admin-v1-DeleteProductRequest) | [DeleteProductResponse](#moth-admin-v1-DeleteProductResponse) |  |

 



<a name="moth_admin_v1_subscription-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/subscription.proto



<a name="moth-admin-v1-GetUserSubscriptionRequest"></a>

### GetUserSubscriptionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| id | [string](#string) |  |  |






<a name="moth-admin-v1-GetUserSubscriptionResponse"></a>

### GetUserSubscriptionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| subscription | [Subscription](#moth-admin-v1-Subscription) |  |  |






<a name="moth-admin-v1-Grant"></a>

### Grant
Grant is one operator (promo/comp) entitlement grant.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| entitlement_id | [string](#string) |  |  |
| expire_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the grant lapses; unset for a non-expiring grant. |
| reason | [string](#string) |  |  |
| granted_by | [string](#string) |  | Label of the operator credential that created it. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| revoke_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Set once revoked. |






<a name="moth-admin-v1-GrantEntitlementRequest"></a>

### GrantEntitlementRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| user_id | [string](#string) |  |  |
| entitlement_id | [string](#string) |  |  |
| expire_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Optional expiry; unset grants indefinitely. |
| reason | [string](#string) |  |  |






<a name="moth-admin-v1-GrantEntitlementResponse"></a>

### GrantEntitlementResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| grant | [Grant](#moth-admin-v1-Grant) |  |  |






<a name="moth-admin-v1-ListUserSubscriptionsRequest"></a>

### ListUserSubscriptionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| user_id | [string](#string) |  |  |






<a name="moth-admin-v1-ListUserSubscriptionsResponse"></a>

### ListUserSubscriptionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| subscriptions | [Subscription](#moth-admin-v1-Subscription) | repeated |  |
| grants | [Grant](#moth-admin-v1-Grant) | repeated | All grants of the user (active, expired and revoked), newest first. |






<a name="moth-admin-v1-RevokeGrantRequest"></a>

### RevokeGrantRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| grant_id | [string](#string) |  |  |






<a name="moth-admin-v1-RevokeGrantResponse"></a>

### RevokeGrantResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| grant | [Grant](#moth-admin-v1-Grant) |  |  |






<a name="moth-admin-v1-Subscription"></a>

### Subscription
Subscription is the operator&#39;s view of one store subscription.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| user_id | [string](#string) |  |  |
| store | [Store](#moth-admin-v1-Store) |  |  |
| product_id | [string](#string) |  | The moth product id the store SKU maps to; empty when unmapped. |
| status | [SubscriptionStatus](#moth-admin-v1-SubscriptionStatus) |  |  |
| current_period_end | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| auto_renew | [bool](#bool) |  |  |
| environment | [string](#string) |  | &#34;sandbox&#34; or &#34;production&#34;. |
| store_transaction_id | [string](#string) |  | Store identity: Apple original_transaction_id or Google purchase_token. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |





 


<a name="moth-admin-v1-Store"></a>

### Store
Store identifies which app store a subscription belongs to.

| Name | Number | Description |
| ---- | ------ | ----------- |
| STORE_UNSPECIFIED | 0 |  |
| STORE_APPLE | 1 |  |
| STORE_GOOGLE | 2 |  |
| STORE_STRIPE | 3 |  |



<a name="moth-admin-v1-SubscriptionStatus"></a>

### SubscriptionStatus
SubscriptionStatus mirrors the store&#39;s renewal state.

| Name | Number | Description |
| ---- | ------ | ----------- |
| SUBSCRIPTION_STATUS_UNSPECIFIED | 0 |  |
| SUBSCRIPTION_STATUS_ACTIVE | 1 |  |
| SUBSCRIPTION_STATUS_TRIALING | 2 |  |
| SUBSCRIPTION_STATUS_IN_GRACE_PERIOD | 3 |  |
| SUBSCRIPTION_STATUS_IN_BILLING_RETRY | 4 |  |
| SUBSCRIPTION_STATUS_PAUSED | 5 |  |
| SUBSCRIPTION_STATUS_EXPIRED | 6 |  |
| SUBSCRIPTION_STATUS_REVOKED | 7 |  |


 

 


<a name="moth-admin-v1-SubscriptionService"></a>

### SubscriptionService
SubscriptionService gives operators the per-user subscription view and the
promo/comp actions: inspect a user&#39;s subscriptions and grants, comp an
entitlement, or revoke a grant. Grant/revoke actions are audit-logged
(milestone 10). All RPCs require an authenticated admin session.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListUserSubscriptions | [ListUserSubscriptionsRequest](#moth-admin-v1-ListUserSubscriptionsRequest) | [ListUserSubscriptionsResponse](#moth-admin-v1-ListUserSubscriptionsResponse) | ListUserSubscriptions returns a user&#39;s store subscriptions and their operator grants (the full billing picture for the user detail page). |
| GetUserSubscription | [GetUserSubscriptionRequest](#moth-admin-v1-GetUserSubscriptionRequest) | [GetUserSubscriptionResponse](#moth-admin-v1-GetUserSubscriptionResponse) | GetUserSubscription returns one subscription by id. |
| GrantEntitlement | [GrantEntitlementRequest](#moth-admin-v1-GrantEntitlementRequest) | [GrantEntitlementResponse](#moth-admin-v1-GrantEntitlementResponse) | GrantEntitlement writes an operator grant (comp a reviewer, extend a grace period, grant a promo), optionally with an expiry. |
| RevokeGrant | [RevokeGrantRequest](#moth-admin-v1-RevokeGrantRequest) | [RevokeGrantResponse](#moth-admin-v1-RevokeGrantResponse) | RevokeGrant marks an operator grant revoked from now on. |

 



<a name="moth_admin_v1_monetization-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/monetization.proto



<a name="moth-admin-v1-GetOfferingRequest"></a>

### GetOfferingRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| offering | [string](#string) |  | Offering tag; empty selects the default offering. |






<a name="moth-admin-v1-GetOfferingResponse"></a>

### GetOfferingResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| offering | [Offering](#moth-admin-v1-Offering) |  |  |






<a name="moth-admin-v1-GetStoreCatalogStatusRequest"></a>

### GetStoreCatalogStatusRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-GetStoreCatalogStatusResponse"></a>

### GetStoreCatalogStatusResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stores | [StoreCatalogStatus](#moth-admin-v1-StoreCatalogStatus) | repeated | One entry per store (Apple, Google). |






<a name="moth-admin-v1-GuidedStep"></a>

### GuidedStep
GuidedStep is a manual action the store APIs cannot perform, stated with the
exact values to enter (honest automation: never silently skipped).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| detail | [string](#string) |  |  |
| url | [string](#string) |  | Deep link into the store console, when one exists. |
| values | [GuidedValue](#moth-admin-v1-GuidedValue) | repeated |  |






<a name="moth-admin-v1-GuidedValue"></a>

### GuidedValue
GuidedValue is one exact value an operator must paste into a store console.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| label | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="moth-admin-v1-Offering"></a>

### Offering
Offering is the ordered set of products a paywall presents (the products
sharing an `offering` tag, in sort_order). Every project has a default
offering (&#34;default&#34;).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [string](#string) |  |  |
| is_default | [bool](#bool) |  |  |
| product_ids | [string](#string) | repeated | Product ids in paywall order. |
| products | [Product](#moth-admin-v1-Product) | repeated | The products themselves, in the same order. |






<a name="moth-admin-v1-ProductSyncItem"></a>

### ProductSyncItem
ProductSyncItem is one product&#39;s line in a sync plan (dry_run) or applied
result.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| product_id | [string](#string) |  |  |
| identifier | [string](#string) |  | The moth catalog identifier (stable; e.g. &#34;monthly&#34;). |
| store_product_id | [string](#string) |  | The store SKU this maps to (desired, or created). |
| action | [SyncAction](#moth-admin-v1-SyncAction) |  |  |
| summary | [string](#string) |  | Human summary of the change (&#34;create monthly at 9.99 USD / month&#34;). |
| changes | [SyncFieldChange](#moth-admin-v1-SyncFieldChange) | repeated | Field-level changes pushed to the store. |
| guided_steps | [GuidedStep](#moth-admin-v1-GuidedStep) | repeated | Manual steps the API cannot perform for this product. |
| status | [ProductSyncStatus](#moth-admin-v1-ProductSyncStatus) |  | Reconcile outcome (unset on dry_run). |
| error | [string](#string) |  | Error when status is ERROR. |






<a name="moth-admin-v1-ProductSyncState"></a>

### ProductSyncState
ProductSyncState is one product&#39;s standing sync record for a store.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| product_id | [string](#string) |  |  |
| identifier | [string](#string) |  |  |
| store_product_id | [string](#string) |  |  |
| status | [ProductSyncStatus](#moth-admin-v1-ProductSyncStatus) |  |  |
| revision | [string](#string) |  | Opaque store-side revision observed at last sync. |
| error | [string](#string) |  |  |
| last_sync_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="moth-admin-v1-ReorderOfferingRequest"></a>

### ReorderOfferingRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| offering | [string](#string) |  | Offering tag; empty selects the default offering. |
| product_ids | [string](#string) | repeated | Product ids in the desired paywall order. Every product currently in the offering must appear exactly once. |






<a name="moth-admin-v1-ReorderOfferingResponse"></a>

### ReorderOfferingResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| offering | [Offering](#moth-admin-v1-Offering) |  |  |






<a name="moth-admin-v1-StoreCatalogStatus"></a>

### StoreCatalogStatus
StoreCatalogStatus is one store&#39;s connection &#43; reconciliation summary.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| store | [Store](#moth-admin-v1-Store) |  |  |
| credentials_present | [bool](#bool) |  | Whether this store&#39;s API credentials are configured. |
| notifications_wired | [bool](#bool) |  | Whether the notification plumbing is wired (Apple ASSN URL / Play RTDN Pub/Sub topic present in the billing config). |
| products_total | [int32](#int32) |  |  |
| products_in_sync | [int32](#int32) |  |  |
| products_drift | [int32](#int32) |  |  |
| products_error | [int32](#int32) |  |  |
| products_unmapped | [int32](#int32) |  | Products with no store SKU set for this store. |
| last_sync_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Latest successful sync across the store&#39;s products. |
| products | [ProductSyncState](#moth-admin-v1-ProductSyncState) | repeated | Per-product detail. |






<a name="moth-admin-v1-SyncFieldChange"></a>

### SyncFieldChange
SyncFieldChange is one field the reconcile will (or would) push to the store.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| field | [string](#string) |  | Field path (&#34;price&#34;, &#34;billing_period&#34;, &#34;localization.en-US.name&#34;, …). |
| current | [string](#string) |  | Store-side current value (&#34;&#34; when creating). |
| desired | [string](#string) |  | Desired value from moth&#39;s catalog. |






<a name="moth-admin-v1-SyncStoreCatalogRequest"></a>

### SyncStoreCatalogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| store | [Store](#moth-admin-v1-Store) |  | Which store to reconcile. |
| dry_run | [bool](#bool) |  | When true, compute and return the plan without mutating the store. |






<a name="moth-admin-v1-SyncStoreCatalogResponse"></a>

### SyncStoreCatalogResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| store | [Store](#moth-admin-v1-Store) |  |  |
| items | [ProductSyncItem](#moth-admin-v1-ProductSyncItem) | repeated | Per-product plan (dry_run) or applied result. |
| in_sync | [bool](#bool) |  | True when the catalog already matches the store (every item is noop) — the idempotency signal. |
| guided_steps | [GuidedStep](#moth-admin-v1-GuidedStep) | repeated | Guided steps not tied to a single product (e.g. notification wiring). |





 


<a name="moth-admin-v1-ProductSyncStatus"></a>

### ProductSyncStatus
ProductSyncStatus mirrors the store.product_store_sync.status column: the
standing reconciliation state of one product for one store.

| Name | Number | Description |
| ---- | ------ | ----------- |
| PRODUCT_SYNC_STATUS_UNSPECIFIED | 0 |  |
| PRODUCT_SYNC_STATUS_PENDING | 1 | Never pushed to this store. |
| PRODUCT_SYNC_STATUS_IN_SYNC | 2 | Store SKU matches moth. |
| PRODUCT_SYNC_STATUS_DRIFT | 3 | Store SKU exists but differs from moth. |
| PRODUCT_SYNC_STATUS_ERROR | 4 | Last reconcile failed. |



<a name="moth-admin-v1-SyncAction"></a>

### SyncAction
SyncAction is the reconcile verb the plan assigns to one product for one
store.

| Name | Number | Description |
| ---- | ------ | ----------- |
| SYNC_ACTION_UNSPECIFIED | 0 |  |
| SYNC_ACTION_NOOP | 1 | The store SKU already matches moth&#39;s desired state. |
| SYNC_ACTION_CREATE | 2 | Create the product (subscription/base plan) in the store. |
| SYNC_ACTION_UPDATE | 3 | Update store-side values that drifted (e.g. a changed price). |
| SYNC_ACTION_GUIDED | 4 | The store API cannot perform this; guided_steps carry the exact manual actions and values. |


 

 


<a name="moth-admin-v1-MonetizationService"></a>

### MonetizationService
MonetizationService reconciles a project&#39;s product catalog (milestone 11 —
entitlements, products, offering) into App Store Connect and Google Play, and
reports the reconciliation state. moth&#39;s catalog is the desired state; the
stores stay authoritative for money and renewals, so a push produces a
diff/plan a human can inspect and any step the store APIs cannot perform is
surfaced as a guided checklist rather than silently skipped.

CAPABILITY SPIKE — what each store&#39;s official API actually allows (the
milestone-08 &#34;honest automation&#34; contract: automate what the APIs expose,
guided fallback with exact paste values where they don&#39;t, verify the result):

  Apple, App Store Connect API v1 (ASC JWT auth from milestone 08):
    AUTOMATABLE: create subscriptionGroups; create auto-renewable
      subscriptions; set base subscriptionPrices from price points; create
      in-app-purchase/subscription localizations (display name &#43; description);
      read created product ids back to map onto moth products.
    GUIDED (API cannot, or cannot reliably, do it): price-schedule nuances and
      regional availability, and — the canonical example — SUBMITTING A NEW
      SUBSCRIPTION FOR REVIEW. These become guided steps with exact values.
    NOTIFICATIONS: the App Store Server Notifications V2 URL is set on the app
      version / in App Store Connect; moth supplies the exact URL &#43; shared
      secret to register (guided), then verifies via a sandbox status read.

  Google, Android Publisher API v3 (service-account authed):
    AUTOMATABLE: monetization.subscriptions with basePlans &#43; offers (price,
      billing period, free-trial/intro phases); read ids back. inappproducts
      is the legacy path for one-off SKUs.
    GUIDED: Play Console product activation/availability and country pricing
      matrices the API leaves to the console.
    NOTIFICATIONS: create/point a Cloud Pub/Sub topic for Real-time developer
      notifications and register it on the app; moth points the push
      subscription at /billing/google/rtdn/{slug}. Topic creation lives in
      GCP (Pub/Sub API), not the Publisher API — guided where credentials
      don&#39;t cover it.

All RPCs require an authenticated admin session. The store API calls
themselves live behind injectable clients (a later stage); this service is
the contract they satisfy.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SyncStoreCatalog | [SyncStoreCatalogRequest](#moth-admin-v1-SyncStoreCatalogRequest) | [SyncStoreCatalogResponse](#moth-admin-v1-SyncStoreCatalogResponse) | SyncStoreCatalog reconciles moth&#39;s catalog into one store. With dry_run it returns the plan (per product: noop | create | update | guided) without mutating the store; otherwise it applies the plan and returns the result. Idempotent by contract: a second run with no catalog change reports in_sync with an all-noop plan. |
| GetStoreCatalogStatus | [GetStoreCatalogStatusRequest](#moth-admin-v1-GetStoreCatalogStatusRequest) | [GetStoreCatalogStatusResponse](#moth-admin-v1-GetStoreCatalogStatusResponse) | GetStoreCatalogStatus returns, per store, whether credentials and notification plumbing are configured and a catalog drift summary — the data behind the admin store-connection panel and `moth doctor` billing checks. |
| GetOffering | [GetOfferingRequest](#moth-admin-v1-GetOfferingRequest) | [GetOfferingResponse](#moth-admin-v1-GetOfferingResponse) | GetOffering returns an offering — the ordered set of products a paywall presents (the products sharing an `offering` tag, in sort_order). An empty offering id selects the default offering. |
| ReorderOffering | [ReorderOfferingRequest](#moth-admin-v1-ReorderOfferingRequest) | [ReorderOfferingResponse](#moth-admin-v1-ReorderOfferingResponse) | ReorderOffering rewrites the paywall order of an offering&#39;s products. Every product currently in the offering must appear exactly once. |

 



<a name="moth_admin_v1_paywall-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/paywall.proto



<a name="moth-admin-v1-GetPaywallConfigRequest"></a>

### GetPaywallConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-GetPaywallConfigResponse"></a>

### GetPaywallConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config | [PaywallConfig](#moth-admin-v1-PaywallConfig) |  |  |
| revision_id | [string](#string) |  | Empty when the project renders the built-in default paywall config. |
| is_default | [bool](#bool) |  | True when no config was ever saved (or after ResetPaywall). |






<a name="moth-admin-v1-ListPaywallRevisionsRequest"></a>

### ListPaywallRevisionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| limit | [int32](#int32) |  | Maximum revisions to return; 0 (or anything above what the server keeps) returns all kept revisions. |






<a name="moth-admin-v1-ListPaywallRevisionsResponse"></a>

### ListPaywallRevisionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revisions | [PaywallRevision](#moth-admin-v1-PaywallRevision) | repeated | Newest first. |






<a name="moth-admin-v1-PaywallConfig"></a>

### PaywallConfig
PaywallConfig is one project&#39;s complete paywall configuration, mirroring the
versioned JSON schema in internal/paywall (the schema version is a storage
concern and does not appear here).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| headline | [string](#string) |  | Primary title (required, bounded length). |
| subtitle | [string](#string) |  | Supporting line under the headline (optional, bounded length). |
| benefits | [string](#string) | repeated | Feature/benefit bullets, in display order (bounded count and length). |
| offering | [string](#string) |  | The offering tag whose products the paywall lists; empty selects the project&#39;s default offering. |
| highlighted_product_identifier | [string](#string) |  | The product identifier rendered as &#34;most popular&#34;; empty highlights nothing. A stable catalog identifier (e.g. &#34;yearly&#34;), never a store SKU. |
| layout | [PaywallLayout](#moth-admin-v1-PaywallLayout) |  |  |
| legal | [PaywallLegal](#moth-admin-v1-PaywallLegal) |  |  |






<a name="moth-admin-v1-PaywallLegal"></a>

### PaywallLegal
PaywallLegal holds the optional legal links rendered in the paywall footer;
must be absolute http(s) URLs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| terms_url | [string](#string) |  |  |
| privacy_url | [string](#string) |  |  |






<a name="moth-admin-v1-PaywallRevision"></a>

### PaywallRevision
PaywallRevision is one saved version of a project&#39;s paywall config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revision_id | [string](#string) |  |  |
| config | [PaywallConfig](#moth-admin-v1-PaywallConfig) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="moth-admin-v1-ResetPaywallRequest"></a>

### ResetPaywallRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-ResetPaywallResponse"></a>

### ResetPaywallResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config | [PaywallConfig](#moth-admin-v1-PaywallConfig) |  | The built-in default paywall config now in effect. |






<a name="moth-admin-v1-RestorePaywallRevisionRequest"></a>

### RestorePaywallRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| revision_id | [string](#string) |  |  |






<a name="moth-admin-v1-RestorePaywallRevisionResponse"></a>

### RestorePaywallRevisionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config | [PaywallConfig](#moth-admin-v1-PaywallConfig) |  |  |
| revision_id | [string](#string) |  | The id of the new revision created by the restore. |






<a name="moth-admin-v1-UpdatePaywallConfigRequest"></a>

### UpdatePaywallConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| config | [PaywallConfig](#moth-admin-v1-PaywallConfig) |  | The full replacement config. |






<a name="moth-admin-v1-UpdatePaywallConfigResponse"></a>

### UpdatePaywallConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config | [PaywallConfig](#moth-admin-v1-PaywallConfig) |  |  |
| revision_id | [string](#string) |  | The id of the revision this save created. |





 


<a name="moth-admin-v1-PaywallLayout"></a>

### PaywallLayout
PaywallLayout is the rendering variant the paywall screen uses; the token
space (colors/spacing/radius) always comes from the theme.

| Name | Number | Description |
| ---- | ------ | ----------- |
| PAYWALL_LAYOUT_UNSPECIFIED | 0 |  |
| PAYWALL_LAYOUT_TILES | 1 | One card per tier, side by side (the default). |
| PAYWALL_LAYOUT_LIST | 2 | Tiers stacked as full-width rows. |
| PAYWALL_LAYOUT_COMPACT | 3 | A single selected tier with a period toggle. |


 

 


<a name="moth-admin-v1-PaywallService"></a>

### PaywallService
PaywallService manages a project&#39;s paywall configuration (milestone 13): the
copy and layout the SDK&#39;s batteries-included paywall screen renders from.
The paywall owns no design tokens of its own — colors, typography, spacing,
radius and logo inherit from the theme (ThemeService, milestone 06); this
config only carries the headline/subtitle copy, benefit bullets, which
offering to present, which tier to highlight, the layout variant and the
legal links. Configs are validated server-side (bounded copy, known layout,
http(s) legal links), and every save creates a revision (the last 10 are
kept for undo), mirroring ThemeService exactly. All RPCs require an
authenticated admin session.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetPaywallConfig | [GetPaywallConfigRequest](#moth-admin-v1-GetPaywallConfigRequest) | [GetPaywallConfigResponse](#moth-admin-v1-GetPaywallConfigResponse) | GetPaywallConfig returns the project&#39;s current paywall config: the saved one, or the built-in defaults when the project never customized anything (revision_id empty, is_default true). |
| UpdatePaywallConfig | [UpdatePaywallConfigRequest](#moth-admin-v1-UpdatePaywallConfigRequest) | [UpdatePaywallConfigResponse](#moth-admin-v1-UpdatePaywallConfigResponse) | UpdatePaywallConfig validates and installs a full replacement config and returns the new revision. Partial updates are done client-side: GetPaywallConfig, edit, UpdatePaywallConfig. |
| ListPaywallRevisions | [ListPaywallRevisionsRequest](#moth-admin-v1-ListPaywallRevisionsRequest) | [ListPaywallRevisionsResponse](#moth-admin-v1-ListPaywallRevisionsResponse) | ListPaywallRevisions returns the saved revisions, newest first (at most the 10 the server keeps). |
| RestorePaywallRevision | [RestorePaywallRevisionRequest](#moth-admin-v1-RestorePaywallRevisionRequest) | [RestorePaywallRevisionResponse](#moth-admin-v1-RestorePaywallRevisionResponse) | RestorePaywallRevision re-installs an old revision&#39;s config as a new revision (history only ever moves forward). |
| ResetPaywall | [ResetPaywallRequest](#moth-admin-v1-ResetPaywallRequest) | [ResetPaywallResponse](#moth-admin-v1-ResetPaywallResponse) | ResetPaywall reverts the project to the built-in default paywall config. The revision history is kept, so the previous config stays restorable. |

 



<a name="moth_admin_v1_theme-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/theme.proto



<a name="moth-admin-v1-DeleteLogoRequest"></a>

### DeleteLogoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| variant | [LogoVariant](#moth-admin-v1-LogoVariant) |  |  |






<a name="moth-admin-v1-DeleteLogoResponse"></a>

### DeleteLogoResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| theme | [Theme](#moth-admin-v1-Theme) |  |  |
| revision_id | [string](#string) |  | The id of the revision this delete created. |






<a name="moth-admin-v1-GetThemeRequest"></a>

### GetThemeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-GetThemeResponse"></a>

### GetThemeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| theme | [Theme](#moth-admin-v1-Theme) |  |  |
| revision_id | [string](#string) |  | Empty when the project renders the built-in defaults. |
| is_default | [bool](#bool) |  | True when no theme was ever saved (or after ResetTheme). |






<a name="moth-admin-v1-ListThemeRevisionsRequest"></a>

### ListThemeRevisionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| limit | [int32](#int32) |  | Maximum revisions to return; 0 (or anything above what the server keeps) returns all kept revisions. |






<a name="moth-admin-v1-ListThemeRevisionsResponse"></a>

### ListThemeRevisionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revisions | [ThemeRevision](#moth-admin-v1-ThemeRevision) | repeated | Newest first. |






<a name="moth-admin-v1-ResetThemeRequest"></a>

### ResetThemeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-ResetThemeResponse"></a>

### ResetThemeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| theme | [Theme](#moth-admin-v1-Theme) |  | The built-in default theme now in effect. |






<a name="moth-admin-v1-RestoreThemeRevisionRequest"></a>

### RestoreThemeRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| revision_id | [string](#string) |  |  |






<a name="moth-admin-v1-RestoreThemeRevisionResponse"></a>

### RestoreThemeRevisionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| theme | [Theme](#moth-admin-v1-Theme) |  |  |
| revision_id | [string](#string) |  | The id of the new revision created by the restore. |






<a name="moth-admin-v1-Theme"></a>

### Theme
Theme is one project&#39;s complete design system, mirroring the versioned
JSON schema in internal/theme (the schema version is a storage concern
and does not appear here).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| colors | [ThemeColors](#moth-admin-v1-ThemeColors) |  | Light palette. Every field is required, &#34;#RRGGBB&#34;. |
| dark_colors | [ThemeColorOverrides](#moth-admin-v1-ThemeColorOverrides) |  | Optional per-field dark-palette overrides; empty fields (or the whole message) are derived from `colors`: surfaces blend toward black, accents toward white, and each on* color becomes black or white, whichever contrasts more. |
| typography | [ThemeTypography](#moth-admin-v1-ThemeTypography) |  |  |
| spacing | [ThemeSpacing](#moth-admin-v1-ThemeSpacing) |  |  |
| shape | [ThemeShape](#moth-admin-v1-ThemeShape) |  |  |
| logo | [ThemeLogo](#moth-admin-v1-ThemeLogo) |  | Output only: logo asset paths are managed through UploadLogo / DeleteLogo; values sent in UpdateTheme are ignored. |
| legal | [ThemeLegal](#moth-admin-v1-ThemeLegal) |  |  |






<a name="moth-admin-v1-ThemeColorOverrides"></a>

### ThemeColorOverrides
ThemeColorOverrides is a partial dark palette; empty fields are derived.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| primary | [string](#string) |  |  |
| on_primary | [string](#string) |  |  |
| background | [string](#string) |  |  |
| on_background | [string](#string) |  |  |
| surface | [string](#string) |  |  |
| on_surface | [string](#string) |  |  |
| error | [string](#string) |  |  |
| on_error | [string](#string) |  |  |






<a name="moth-admin-v1-ThemeColors"></a>

### ThemeColors
ThemeColors is a complete palette: each color role and its &#34;on&#34;
(foreground) counterpart. Validation enforces a WCAG AA contrast ratio
(&gt;= 4.5:1) between every pair.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| primary | [string](#string) |  |  |
| on_primary | [string](#string) |  |  |
| background | [string](#string) |  |  |
| on_background | [string](#string) |  |  |
| surface | [string](#string) |  |  |
| on_surface | [string](#string) |  |  |
| error | [string](#string) |  |  |
| on_error | [string](#string) |  |  |






<a name="moth-admin-v1-ThemeLegal"></a>

### ThemeLegal
ThemeLegal holds the optional legal links rendered near signup; must be
absolute http(s) URLs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| terms_url | [string](#string) |  |  |
| privacy_url | [string](#string) |  |  |






<a name="moth-admin-v1-ThemeLogo"></a>

### ThemeLogo
ThemeLogo holds the server-managed asset paths of the uploaded logos
(&#34;/assets/{project}/logo-light.png&#34;); empty means no logo.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| light_path | [string](#string) |  |  |
| dark_path | [string](#string) |  |  |






<a name="moth-admin-v1-ThemeRevision"></a>

### ThemeRevision
ThemeRevision is one saved version of a project&#39;s theme.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revision_id | [string](#string) |  |  |
| theme | [Theme](#moth-admin-v1-Theme) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="moth-admin-v1-ThemeShape"></a>

### ThemeShape



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| corner_radius | [int32](#int32) |  | Component corner radius in logical pixels, 0 to 32. |






<a name="moth-admin-v1-ThemeSpacing"></a>

### ThemeSpacing



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| unit | [int32](#int32) |  | Base spacing step in logical pixels, 4 to 16. |






<a name="moth-admin-v1-ThemeTypography"></a>

### ThemeTypography



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| font_family | [string](#string) |  | One of the curated embedded fonts (e.g. &#34;Inter&#34;); arbitrary fonts are rejected. |
| scale | [double](#double) |  | Global text-size multiplier, 0.8 to 1.4. |






<a name="moth-admin-v1-UpdateThemeRequest"></a>

### UpdateThemeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| theme | [Theme](#moth-admin-v1-Theme) |  | The full replacement token set (logo paths excepted, see Theme.logo). |






<a name="moth-admin-v1-UpdateThemeResponse"></a>

### UpdateThemeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| theme | [Theme](#moth-admin-v1-Theme) |  |  |
| revision_id | [string](#string) |  | The id of the revision this save created. |






<a name="moth-admin-v1-UploadLogoRequest"></a>

### UploadLogoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| variant | [LogoVariant](#moth-admin-v1-LogoVariant) |  |  |
| data | [bytes](#bytes) |  | Image bytes; PNG (&#34;image/png&#34;) or SVG (&#34;image/svg&#43;xml&#34;), at most 512 KiB. |
| content_type | [string](#string) |  |  |






<a name="moth-admin-v1-UploadLogoResponse"></a>

### UploadLogoResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| theme | [Theme](#moth-admin-v1-Theme) |  |  |
| revision_id | [string](#string) |  | The id of the revision this upload created. |
| path | [string](#string) |  | Server path of the stored asset (&#34;/assets/{project}/logo-light.png&#34;). |





 


<a name="moth-admin-v1-LogoVariant"></a>

### LogoVariant
LogoVariant selects which color scheme a logo is for.

| Name | Number | Description |
| ---- | ------ | ----------- |
| LOGO_VARIANT_UNSPECIFIED | 0 |  |
| LOGO_VARIANT_LIGHT | 1 |  |
| LOGO_VARIANT_DARK | 2 |  |


 

 


<a name="moth-admin-v1-ThemeService"></a>

### ThemeService
ThemeService manages a project&#39;s design system: the small token set
(colors, typography, spacing, corner radius, logo, legal links) that
every end-user surface renders from. Themes are validated server-side —
WCAG AA contrast on every color/on-color pair, curated fonts, bounded
ranges — so any accepted theme produces a legible screen. Every save
creates a revision (the last 10 are kept for undo). All RPCs require an
authenticated admin session.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetTheme | [GetThemeRequest](#moth-admin-v1-GetThemeRequest) | [GetThemeResponse](#moth-admin-v1-GetThemeResponse) | GetTheme returns the project&#39;s current theme: the saved one, or the built-in defaults when the project never customized anything (revision_id empty, is_default true). |
| UpdateTheme | [UpdateThemeRequest](#moth-admin-v1-UpdateThemeRequest) | [UpdateThemeResponse](#moth-admin-v1-UpdateThemeResponse) | UpdateTheme validates and installs a full replacement token set and returns the new revision. Partial updates are done client-side: GetTheme, edit, UpdateTheme. |
| ListThemeRevisions | [ListThemeRevisionsRequest](#moth-admin-v1-ListThemeRevisionsRequest) | [ListThemeRevisionsResponse](#moth-admin-v1-ListThemeRevisionsResponse) | ListThemeRevisions returns the saved revisions, newest first (at most the 10 the server keeps). |
| RestoreThemeRevision | [RestoreThemeRevisionRequest](#moth-admin-v1-RestoreThemeRevisionRequest) | [RestoreThemeRevisionResponse](#moth-admin-v1-RestoreThemeRevisionResponse) | RestoreThemeRevision re-installs an old revision&#39;s token set as a new revision (history only ever moves forward). Note that logo assets are stored per project, not per revision: a restored theme points at the logo files as they are today. |
| ResetTheme | [ResetThemeRequest](#moth-admin-v1-ResetThemeRequest) | [ResetThemeResponse](#moth-admin-v1-ResetThemeResponse) | ResetTheme reverts the project to the built-in default theme. The revision history is kept, so the previous theme stays restorable. |
| UploadLogo | [UploadLogoRequest](#moth-admin-v1-UploadLogoRequest) | [UploadLogoResponse](#moth-admin-v1-UploadLogoResponse) | UploadLogo stores a logo image for one color scheme and returns the updated theme (a new revision). PNG or SVG, at most 512 KiB; images are decoded and re-encoded server-side, which strips any embedded payloads (SVG scripts in particular). |
| DeleteLogo | [DeleteLogoRequest](#moth-admin-v1-DeleteLogoRequest) | [DeleteLogoResponse](#moth-admin-v1-DeleteLogoResponse) | DeleteLogo removes one color scheme&#39;s logo from the theme (a new revision) and deletes the stored file. |

 



<a name="moth_admin_v1_project-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/project.proto



<a name="moth-admin-v1-AppleProviderConfig"></a>

### AppleProviderConfig
AppleProviderConfig configures Sign in with Apple for one project.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| services_id | [string](#string) |  | Services ID (web-redirect flow `aud`, e.g. &#34;com.example.app.signin&#34;). |
| team_id | [string](#string) |  | Apple Developer Team ID. |
| key_id | [string](#string) |  | Key ID of the &#34;Sign in with Apple&#34; private key. |
| private_key_p8 | [string](#string) |  | Contents of the `.p8` private key file, used to mint Apple client secrets. Write-only: accepted on update, never returned by reads. On update, an empty value keeps the stored one; reads report presence via has_private_key. Stored encrypted at rest. |
| has_private_key | [bool](#bool) |  | Output only: whether a private key is stored. |
| bundle_ids | [string](#string) | repeated | App bundle IDs accepted as `aud` on native Apple ID tokens. |






<a name="moth-admin-v1-CreateProjectRequest"></a>

### CreateProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| slug | [string](#string) |  | Optional explicit slug (lowercase letters, digits and single dashes). Empty derives one from the name, adding a suffix on collision; an explicit slug that is already taken fails ALREADY_EXISTS. Lets `moth project apply` create a project under the exact slug its spec is keyed on. |






<a name="moth-admin-v1-CreateProjectResponse"></a>

### CreateProjectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#moth-admin-v1-Project) |  |  |
| secret_key | [string](#string) |  | The server-to-server secret key. Stored hashed server-side and therefore returned exactly once, in this response. |






<a name="moth-admin-v1-DeleteProjectRequest"></a>

### DeleteProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="moth-admin-v1-DeleteProjectResponse"></a>

### DeleteProjectResponse







<a name="moth-admin-v1-EntitlementSpec"></a>

### EntitlementSpec
EntitlementSpec is one named capability. `identifier` is the identity apply
keys on (immutable); only display_name is editable.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [string](#string) |  |  |
| display_name | [string](#string) |  |  |






<a name="moth-admin-v1-ExportProjectRequest"></a>

### ExportProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-ExportProjectResponse"></a>

### ExportProjectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| users | [ExportedUser](#moth-admin-v1-ExportedUser) | repeated |  |






<a name="moth-admin-v1-ExportedIdentity"></a>

### ExportedIdentity
ExportedIdentity is one linked provider of an exported/imported user.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  | &#34;password&#34;, &#34;google&#34; or &#34;apple&#34;. |
| provider_subject | [string](#string) |  | Provider-issued subject (the user id for password identities). |
| email | [string](#string) |  | Email the provider asserted when the identity was linked. |






<a name="moth-admin-v1-ExportedUser"></a>

### ExportedUser
ExportedUser is one user in an export document, with everything needed to
recreate the account on another system.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| email | [string](#string) |  |  |
| email_verified | [bool](#bool) |  |  |
| display_name | [string](#string) |  |  |
| avatar_url | [string](#string) |  |  |
| custom_claims | [string](#string) |  | JSON object embedded in the JWT `claims` claim. |
| disabled | [bool](#bool) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| last_login_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| password_hash | [string](#string) |  | Encoded password hash; empty for social-only accounts. |
| password_algorithm | [string](#string) |  | Algorithm that produced password_hash: &#34;argon2id&#34; for a native moth hash, or the foreign algorithm it was imported with. |
| identities | [ExportedIdentity](#moth-admin-v1-ExportedIdentity) | repeated |  |






<a name="moth-admin-v1-GetProjectRequest"></a>

### GetProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |






<a name="moth-admin-v1-GetProjectResponse"></a>

### GetProjectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#moth-admin-v1-Project) |  |  |






<a name="moth-admin-v1-GetSigningKeyRequest"></a>

### GetSigningKeyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-GetSigningKeyResponse"></a>

### GetSigningKeyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [SigningKey](#moth-admin-v1-SigningKey) |  |  |
| jwks_url | [string](#string) |  | Absolute URL of the project JWKS document. |
| issuer | [string](#string) |  | Expected `iss` claim of this project&#39;s access tokens. |
| audience | [string](#string) |  | Expected `aud` claim (the project slug). |






<a name="moth-admin-v1-GoogleProviderConfig"></a>

### GoogleProviderConfig
GoogleProviderConfig configures Sign in with Google for one project. The
client IDs are the allowed `aud` values when verifying Google ID tokens.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| web_client_id | [string](#string) |  | OAuth client ID of type &#34;Web application&#34;; also used by the web-redirect fallback flow. |
| ios_client_id | [string](#string) |  | OAuth client ID of type &#34;iOS&#34;. |
| android_client_id | [string](#string) |  | OAuth client ID of type &#34;Android&#34;. |
| web_client_secret | [string](#string) |  | Client secret of the web client, needed for the web-redirect code exchange. Write-only: accepted on update, never returned by reads. On update, an empty value keeps the stored one; reads report presence via has_web_client_secret. Stored encrypted at rest. |
| has_web_client_secret | [bool](#bool) |  | Output only: whether a web client secret is stored. |






<a name="moth-admin-v1-ImportProjectRequest"></a>

### ImportProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| users | [ImportedUser](#moth-admin-v1-ImportedUser) | repeated |  |






<a name="moth-admin-v1-ImportProjectResponse"></a>

### ImportProjectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| imported_count | [int32](#int32) |  | Number of users actually created. |
| skipped_count | [int32](#int32) |  | Number skipped because their email was already registered. |






<a name="moth-admin-v1-ImportedUser"></a>

### ImportedUser
ImportedUser is one user to create, optionally carrying a foreign password
hash. A foreign hash is verified with its original algorithm on the user&#39;s
first sign-in, then transparently rehashed to argon2id.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |
| email_verified | [bool](#bool) |  |  |
| display_name | [string](#string) |  |  |
| avatar_url | [string](#string) |  |  |
| custom_claims | [string](#string) |  | JSON object embedded in the JWT `claims` claim (defaults to &#34;{}&#34;). |
| password_hash | [string](#string) |  | Encoded password hash; empty for users without a password. |
| password_algorithm | [string](#string) |  | Algorithm that produced password_hash: &#34;bcrypt&#34;, &#34;scrypt&#34;, &#34;argon2&#34; or &#34;pbkdf2&#34; for a foreign hash, or &#34;argon2id&#34;/&#34;&#34; for a native moth hash. |
| identities | [ExportedIdentity](#moth-admin-v1-ExportedIdentity) | repeated |  |
| disabled | [bool](#bool) |  | Whether the account is created disabled (blocked from signing in). |






<a name="moth-admin-v1-ListProjectsRequest"></a>

### ListProjectsRequest







<a name="moth-admin-v1-ListProjectsResponse"></a>

### ListProjectsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | [Project](#moth-admin-v1-Project) | repeated |  |






<a name="moth-admin-v1-MonetizationSpec"></a>

### MonetizationSpec
MonetizationSpec is a project&#39;s declarative subscription catalog: the
entitlements apps gate on and the products (tiers) that grant them. Keyed on
identifiers (not server-assigned ids) so it round-trips through dump/apply.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entitlements | [EntitlementSpec](#moth-admin-v1-EntitlementSpec) | repeated |  |
| products | [ProductSpec](#moth-admin-v1-ProductSpec) | repeated |  |






<a name="moth-admin-v1-ProductSpec"></a>

### ProductSpec
ProductSpec is one subscription tier in declarative form. `identifier` is the
identity apply keys on. `entitlements` and `offering`/`sort_order` capture the
grant set and paywall placement; an offering is just the set of products
sharing an `offering` tag, ordered by sort_order.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [string](#string) |  |  |
| display_name | [string](#string) |  |  |
| apple_product_id | [string](#string) |  | Store SKUs; either may be empty when the tier ships on one store only. |
| google_product_id | [string](#string) |  |  |
| billing_period | [string](#string) |  | Billing period (&#34;weekly&#34;, &#34;monthly&#34;, &#34;yearly&#34;, …); the store read is authoritative. |
| price_amount_micros | [int64](#int64) |  | List price metadata (display &#43; analytics only). Micros = price * 1_000_000. |
| currency | [string](#string) |  |  |
| trial_period | [string](#string) |  |  |
| intro_price_amount_micros | [int64](#int64) |  |  |
| intro_period | [string](#string) |  |  |
| offering | [string](#string) |  | Paywall grouping tag (&#34;default&#34; when empty) and order within it. |
| sort_order | [int32](#int32) |  |  |
| entitlements | [string](#string) | repeated | Entitlement IDENTIFIERS (not ids) this product grants while active. |






<a name="moth-admin-v1-Project"></a>

### Project



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| slug | [string](#string) |  | URL-safe identifier, used in hosted-page and JWKS URLs (/p/{slug}/...). |
| publishable_key | [string](#string) |  | Identifies the project to the mobile SDK. Safe to embed in an app. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| settings | [ProjectSettings](#moth-admin-v1-ProjectSettings) |  |  |
| user_count | [int64](#int64) |  | Number of end-user accounts in the project. |






<a name="moth-admin-v1-ProjectSettings"></a>

### ProjectSettings
ProjectSettings is the per-project auth policy.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| password_min_length | [int32](#int32) |  | Minimum accepted password length (default 8). |
| require_email_verification | [bool](#bool) |  | Block SignIn until the email address is verified. |
| allow_public_signup | [bool](#bool) |  | Whether the public SignUp RPC is open (invite-only projects: false). |
| enumeration_safe_signup | [bool](#bool) |  | SignUp with an already-registered email returns OK and mails the owner instead of erroring, so responses never reveal whether an account exists. |
| access_token_ttl_seconds | [int32](#int32) |  | Access token (JWT) lifetime in seconds (default 900). |
| refresh_token_ttl_days | [int32](#int32) |  | Refresh token sliding window in days (default 30). |
| google | [GoogleProviderConfig](#moth-admin-v1-GoogleProviderConfig) |  | Sign in with Google configuration. |
| apple | [AppleProviderConfig](#moth-admin-v1-AppleProviderConfig) |  | Sign in with Apple configuration. |
| auto_link_verified_email | [bool](#bool) | optional | Link a social identity to an existing account when the provider asserts the same, verified email. The server default is TRUE: `optional` so that &#34;unset&#34; (from clients predating this field) is distinguishable from an explicit false and reads as the default. Reads always populate it. |
| redirect_schemes | [string](#string) | repeated | Custom URL schemes the web-redirect OAuth fallback may redirect back to (e.g. &#34;myapp&#34;). Open-redirect protection: callbacks only ever redirect to a scheme on this list. |
| analytics_retention_days | [int32](#int32) |  | How long raw analytics events are kept before the rollup job prunes them, in days (default 90). |
| rollup_timezone | [string](#string) |  | IANA timezone name (e.g. &#34;Europe/Paris&#34;) the analytics rollup buckets days in (default &#34;UTC&#34;). |
| signup_email_allowlist | [string](#string) | repeated | When non-empty, signup is restricted to email addresses whose domain matches one of these glob patterns (e.g. &#34;example.com&#34;, &#34;*.acme.io&#34;); every other domain is rejected. |
| signup_email_blocklist | [string](#string) | repeated | Email-domain glob patterns rejected at signup, evaluated after the allowlist. |
| captcha_verify_url | [string](#string) |  | Optional CAPTCHA verification endpoint. The CAPTCHA hook is documented but off by default in v1: this field is stored but not yet wired. |
| redirect_origins | [string](#string) | repeated | Web origins (e.g. &#34;https://app.example.com&#34;) the web-redirect OAuth fallback may redirect back to, for browser SPAs. Exact-origin matching (scheme &#43; host &#43; port) constrains the host, so unlike a bare http(s) scheme this is not an open redirect. Bare origins only — no path, query or fragment; http is accepted for localhost during development. |






<a name="moth-admin-v1-ProjectSpec"></a>

### ProjectSpec
ProjectSpec is the full desired state of one project: the document
`moth project dump` emits and `moth project apply` consumes (serialized
as JSON/YAML via protojson). It reuses this package&#39;s messages verbatim,
so no apply RPC is needed server-side — the CLI composes the existing
calls, keyed on the slug: CreateProject (when no project has the slug)
or UpdateProject (name &#43; settings), then ThemeService.UpdateTheme.
Write-only provider secrets never appear in dumps (reads only report
has_* presence) and an empty secret field on apply keeps the stored
value, matching UpdateProject semantics. Output-only fields (has_*,
logo asset paths) are ignored on apply.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| slug | [string](#string) |  | Identity of the spec: apply creates the project when no project has this slug and updates it otherwise. |
| settings | [ProjectSettings](#moth-admin-v1-ProjectSettings) |  |  |
| theme | [Theme](#moth-admin-v1-Theme) |  | Design-system theme (legal links included); unset means the built-in default theme. |
| monetization | [MonetizationSpec](#moth-admin-v1-MonetizationSpec) |  | Subscription catalog (entitlements, products, offering) — the monetization desired state milestone 12 defines once and reconciles into the stores. Unlike settings (which merge), the catalog is full desired state: apply creates/updates/deletes to match, keyed on stable identifiers, so a dump re-applied diffs to zero. Unset leaves the catalog untouched. Store API credentials are NOT part of the spec — they are write-only secrets managed through BillingCredentialsService / `moth setup billing`. |






<a name="moth-admin-v1-RegenerateSecretKeyRequest"></a>

### RegenerateSecretKeyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-RegenerateSecretKeyResponse"></a>

### RegenerateSecretKeyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#moth-admin-v1-Project) |  |  |
| secret_key | [string](#string) |  | The replacement secret key, returned exactly once. |






<a name="moth-admin-v1-ResetSigningKeyRequest"></a>

### ResetSigningKeyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |






<a name="moth-admin-v1-ResetSigningKeyResponse"></a>

### ResetSigningKeyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [SigningKey](#moth-admin-v1-SigningKey) |  | The replacement signing key. |






<a name="moth-admin-v1-RotateSigningKeyRequest"></a>

### RotateSigningKeyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| grace_seconds | [int32](#int32) |  | Grace period, in seconds, the previous key stays in the JWKS. 0 uses the server default (access-token TTL &#43; clock skew). |






<a name="moth-admin-v1-RotateSigningKeyResponse"></a>

### RotateSigningKeyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [SigningKey](#moth-admin-v1-SigningKey) |  | The new active signing key. |
| grace_expire_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the previous key leaves the JWKS and becomes eligible for pruning. |






<a name="moth-admin-v1-SigningKey"></a>

### SigningKey
SigningKey is the public half of one project token-signing keypair.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kid | [string](#string) |  | RFC 7638 JWK thumbprint, the `kid` on every JWT the key signs. |
| algorithm | [string](#string) |  | Signature algorithm, always &#34;ES256&#34;. |
| public_key_pem | [string](#string) |  | PEM-encoded public key (PKIX), for offline verification setups. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="moth-admin-v1-UpdateProjectRequest"></a>

### UpdateProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| name | [string](#string) |  |  |
| settings | [ProjectSettings](#moth-admin-v1-ProjectSettings) |  | Replaces the whole settings object when set; leave unset to keep the current settings. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | Fields to update (&#34;name&#34;, &#34;settings&#34;). When unset, legacy behavior: name is always applied, settings only when present. |






<a name="moth-admin-v1-UpdateProjectResponse"></a>

### UpdateProjectResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#moth-admin-v1-Project) |  |  |





 

 

 


<a name="moth-admin-v1-ProjectService"></a>

### ProjectService
ProjectService manages the projects (one per mobile app) hosted by this
moth instance. All RPCs require an authenticated admin session.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateProject | [CreateProjectRequest](#moth-admin-v1-CreateProjectRequest) | [CreateProjectResponse](#moth-admin-v1-CreateProjectResponse) |  |
| GetProject | [GetProjectRequest](#moth-admin-v1-GetProjectRequest) | [GetProjectResponse](#moth-admin-v1-GetProjectResponse) |  |
| ListProjects | [ListProjectsRequest](#moth-admin-v1-ListProjectsRequest) | [ListProjectsResponse](#moth-admin-v1-ListProjectsResponse) |  |
| UpdateProject | [UpdateProjectRequest](#moth-admin-v1-UpdateProjectRequest) | [UpdateProjectResponse](#moth-admin-v1-UpdateProjectResponse) |  |
| DeleteProject | [DeleteProjectRequest](#moth-admin-v1-DeleteProjectRequest) | [DeleteProjectResponse](#moth-admin-v1-DeleteProjectResponse) |  |
| RegenerateSecretKey | [RegenerateSecretKeyRequest](#moth-admin-v1-RegenerateSecretKeyRequest) | [RegenerateSecretKeyResponse](#moth-admin-v1-RegenerateSecretKeyResponse) | RegenerateSecretKey replaces the project&#39;s server-to-server secret key. The old key stops working immediately; the new one is returned exactly once, in this response. |
| GetSigningKey | [GetSigningKeyRequest](#moth-admin-v1-GetSigningKeyRequest) | [GetSigningKeyResponse](#moth-admin-v1-GetSigningKeyResponse) | GetSigningKey returns the project&#39;s active token-signing key (public part) and its JWKS URL, for the admin overview card. |
| ResetSigningKey | [ResetSigningKeyRequest](#moth-admin-v1-ResetSigningKeyRequest) | [ResetSigningKeyResponse](#moth-admin-v1-ResetSigningKeyResponse) | ResetSigningKey generates a fresh ES256 keypair, removes every previous key from the project JWKS immediately, and revokes all refresh tokens. Every access token ever issued becomes invalid and all users must sign in again. |
| RotateSigningKey | [RotateSigningKeyRequest](#moth-admin-v1-RotateSigningKeyRequest) | [RotateSigningKeyResponse](#moth-admin-v1-RotateSigningKeyResponse) | RotateSigningKey generates a fresh ES256 keypair that signs new tokens from now on, while the previous key stays in the project JWKS for a grace period (default: access-token TTL &#43; clock skew). Tokens already issued keep validating until they expire, so — unlike ResetSigningKey — no user is signed out. Expired grace keys are pruned automatically. |
| ExportProject | [ExportProjectRequest](#moth-admin-v1-ExportProjectRequest) | [ExportProjectResponse](#moth-admin-v1-ExportProjectResponse) | ExportProject returns the project&#39;s users as JSON for migration off moth (no lock-in). Password hashes are included so accounts can be recreated elsewhere. Large projects should prefer the CLI, which pages. |
| ImportProject | [ImportProjectRequest](#moth-admin-v1-ImportProjectRequest) | [ImportProjectResponse](#moth-admin-v1-ImportProjectResponse) | ImportProject bulk-creates users from a JSON document, optionally carrying foreign password hashes (bcrypt/scrypt/argon2/pbkdf2) so teams can migrate from another provider without a forced password reset. A user whose email already exists is skipped. |

 



<a name="moth_admin_v1_settings-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/settings.proto



<a name="moth-admin-v1-GetInstanceSettingsRequest"></a>

### GetInstanceSettingsRequest







<a name="moth-admin-v1-GetInstanceSettingsResponse"></a>

### GetInstanceSettingsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| base_url | [string](#string) |  | Public base URL of this instance (issuer prefix, JWKS URLs, links). |
| version | [string](#string) |  | Server version (&#34;dev&#34; for unreleased builds). |
| smtp | [SmtpSettings](#moth-admin-v1-SmtpSettings) |  | Effective SMTP settings with the password blanked. |
| smtp_source | [SmtpSource](#moth-admin-v1-SmtpSource) |  |  |
| smtp_has_password | [bool](#bool) |  | Whether the stored/config SMTP has a password set. |






<a name="moth-admin-v1-SendTestEmailRequest"></a>

### SendTestEmailRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| to | [string](#string) |  |  |






<a name="moth-admin-v1-SendTestEmailResponse"></a>

### SendTestEmailResponse







<a name="moth-admin-v1-SmtpSettings"></a>

### SmtpSettings
SmtpSettings is the outgoing email relay configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  |  |
| port | [int32](#int32) |  |  |
| username | [string](#string) |  |  |
| password | [string](#string) |  | Write-only: accepted on update, never returned by reads. On update, an empty password keeps the stored one. |
| from | [string](#string) |  | Sender address on every email. |






<a name="moth-admin-v1-UpdateSmtpSettingsRequest"></a>

### UpdateSmtpSettingsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| smtp | [SmtpSettings](#moth-admin-v1-SmtpSettings) |  |  |






<a name="moth-admin-v1-UpdateSmtpSettingsResponse"></a>

### UpdateSmtpSettingsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| smtp | [SmtpSettings](#moth-admin-v1-SmtpSettings) |  |  |
| smtp_source | [SmtpSource](#moth-admin-v1-SmtpSource) |  |  |
| smtp_has_password | [bool](#bool) |  |  |





 


<a name="moth-admin-v1-SmtpSource"></a>

### SmtpSource
SmtpSource says where the effective SMTP configuration comes from.

| Name | Number | Description |
| ---- | ------ | ----------- |
| SMTP_SOURCE_UNSPECIFIED | 0 |  |
| SMTP_SOURCE_NONE | 1 | No SMTP anywhere; emails are logged to the server console. |
| SMTP_SOURCE_CONFIG | 2 | From the server config file / environment. |
| SMTP_SOURCE_DATABASE | 3 | From the database (set through this service). |


 

 


<a name="moth-admin-v1-InstanceSettingsService"></a>

### InstanceSettingsService
InstanceSettingsService exposes instance-wide configuration to the admin
console: outgoing email and the values the setup-instruction pages
interpolate. All RPCs require an authenticated admin session.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetInstanceSettings | [GetInstanceSettingsRequest](#moth-admin-v1-GetInstanceSettingsRequest) | [GetInstanceSettingsResponse](#moth-admin-v1-GetInstanceSettingsResponse) |  |
| UpdateSmtpSettings | [UpdateSmtpSettingsRequest](#moth-admin-v1-UpdateSmtpSettingsRequest) | [UpdateSmtpSettingsResponse](#moth-admin-v1-UpdateSmtpSettingsResponse) | UpdateSmtpSettings stores an SMTP configuration in the database, which takes precedence over the server config file. An empty host clears the stored configuration and falls back to the config file (or the console transport). |
| SendTestEmail | [SendTestEmailRequest](#moth-admin-v1-SendTestEmailRequest) | [SendTestEmailResponse](#moth-admin-v1-SendTestEmailResponse) | SendTestEmail sends a probe email through the currently effective transport. |

 



<a name="moth_admin_v1_user-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/admin/v1/user.proto



<a name="moth-admin-v1-CreateUserRequest"></a>

### CreateUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| email | [string](#string) |  |  |
| display_name | [string](#string) |  |  |
| password | [string](#string) |  | Initial password; leave empty together with send_invite to let the user choose one through the invite email. |
| email_verified | [bool](#bool) |  | Mark the email address as already verified. |
| send_invite | [bool](#bool) |  | Send a set-password invite email (requires a working mailer). |






<a name="moth-admin-v1-CreateUserResponse"></a>

### CreateUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-admin-v1-User) |  |  |






<a name="moth-admin-v1-DeleteUserRequest"></a>

### DeleteUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| user_id | [string](#string) |  |  |






<a name="moth-admin-v1-DeleteUserResponse"></a>

### DeleteUserResponse







<a name="moth-admin-v1-DisableUserRequest"></a>

### DisableUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| user_id | [string](#string) |  |  |






<a name="moth-admin-v1-DisableUserResponse"></a>

### DisableUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-admin-v1-User) |  |  |






<a name="moth-admin-v1-EnableUserRequest"></a>

### EnableUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| user_id | [string](#string) |  |  |






<a name="moth-admin-v1-EnableUserResponse"></a>

### EnableUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-admin-v1-User) |  |  |






<a name="moth-admin-v1-GetUserRequest"></a>

### GetUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| user_id | [string](#string) |  |  |






<a name="moth-admin-v1-GetUserResponse"></a>

### GetUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-admin-v1-User) |  |  |
| sessions | [UserSession](#moth-admin-v1-UserSession) | repeated |  |
| identities | [Identity](#moth-admin-v1-Identity) | repeated | Linked provider identities, in link order. |






<a name="moth-admin-v1-Identity"></a>

### Identity
Identity is one linked authentication provider of a user, shown on the
user detail page (and driving its unlink action).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [string](#string) |  | &#34;password&#34;, &#34;google&#34; or &#34;apple&#34;. |
| email | [string](#string) |  | Email asserted by the provider when the identity was linked; empty for password identities. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="moth-admin-v1-ListUsersRequest"></a>

### ListUsersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| page_size | [int32](#int32) |  | Page size, 1–200; 0 means the default (50). |
| page_token | [string](#string) |  | next_page_token of the previous response; empty for the first page. |
| query | [string](#string) |  | Case-insensitive substring filter on email and display name. |






<a name="moth-admin-v1-ListUsersResponse"></a>

### ListUsersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| users | [User](#moth-admin-v1-User) | repeated |  |
| next_page_token | [string](#string) |  | Empty when this is the last page. |
| total_size | [int64](#int64) |  | Total users matching the query across all pages. |






<a name="moth-admin-v1-RevokeUserSessionsRequest"></a>

### RevokeUserSessionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| user_id | [string](#string) |  |  |






<a name="moth-admin-v1-RevokeUserSessionsResponse"></a>

### RevokeUserSessionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revoked_count | [int64](#int64) |  | Number of sessions revoked. |






<a name="moth-admin-v1-SendPasswordResetRequest"></a>

### SendPasswordResetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| user_id | [string](#string) |  |  |






<a name="moth-admin-v1-SendPasswordResetResponse"></a>

### SendPasswordResetResponse







<a name="moth-admin-v1-UpdateUserRequest"></a>

### UpdateUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_id | [string](#string) |  |  |
| user_id | [string](#string) |  |  |
| user | [User](#moth-admin-v1-User) |  |  |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | Supported paths: &#34;display_name&#34;, &#34;custom_claims&#34;. |






<a name="moth-admin-v1-UpdateUserResponse"></a>

### UpdateUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-admin-v1-User) |  |  |






<a name="moth-admin-v1-User"></a>

### User
User is the operator&#39;s view of a project end user.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| email | [string](#string) |  |  |
| email_verified | [bool](#bool) |  |  |
| display_name | [string](#string) |  |  |
| disabled | [bool](#bool) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| providers | [string](#string) | repeated | Linked authentication providers (&#34;password&#34;, &#34;google&#34;, &#34;apple&#34;). |
| last_login_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Last successful sign-in; unset when the user never signed in. |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| custom_claims | [string](#string) |  | JSON object embedded in the JWT `claims` claim. |






<a name="moth-admin-v1-UserSession"></a>

### UserSession
UserSession is one active device session (a live refresh-token chain).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| device_info | [string](#string) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| expire_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |





 

 

 


<a name="moth-admin-v1-UserService"></a>

### UserService
UserService gives instance operators visibility and control over a
project&#39;s end users. All RPCs require an authenticated admin session.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListUsers | [ListUsersRequest](#moth-admin-v1-ListUsersRequest) | [ListUsersResponse](#moth-admin-v1-ListUsersResponse) | ListUsers pages through a project&#39;s users, newest first, optionally filtered by a case-insensitive substring match on email or display name. |
| GetUser | [GetUserRequest](#moth-admin-v1-GetUserRequest) | [GetUserResponse](#moth-admin-v1-GetUserResponse) | GetUser returns one user with its provider identities and active sessions. |
| CreateUser | [CreateUserRequest](#moth-admin-v1-CreateUserRequest) | [CreateUserResponse](#moth-admin-v1-CreateUserResponse) | CreateUser adds an account on the operator&#39;s behalf: either with a password, or without one plus an invite email that lets the owner set their own (the counterpart of invite-only signup mode). |
| UpdateUser | [UpdateUserRequest](#moth-admin-v1-UpdateUserRequest) | [UpdateUserResponse](#moth-admin-v1-UpdateUserResponse) | UpdateUser applies the fields named in update_mask (&#34;display_name&#34;, &#34;custom_claims&#34;). |
| DisableUser | [DisableUserRequest](#moth-admin-v1-DisableUserRequest) | [DisableUserResponse](#moth-admin-v1-DisableUserResponse) | DisableUser blocks sign-in, refresh and introspection and revokes the user&#39;s refresh tokens. |
| EnableUser | [EnableUserRequest](#moth-admin-v1-EnableUserRequest) | [EnableUserResponse](#moth-admin-v1-EnableUserResponse) |  |
| DeleteUser | [DeleteUserRequest](#moth-admin-v1-DeleteUserRequest) | [DeleteUserResponse](#moth-admin-v1-DeleteUserResponse) | DeleteUser permanently removes the user, its identities, sessions and pending email tokens. |
| RevokeUserSessions | [RevokeUserSessionsRequest](#moth-admin-v1-RevokeUserSessionsRequest) | [RevokeUserSessionsResponse](#moth-admin-v1-RevokeUserSessionsResponse) | RevokeUserSessions revokes every refresh token of the user (all devices); outstanding access tokens die at their expiry. |
| SendPasswordReset | [SendPasswordResetRequest](#moth-admin-v1-SendPasswordResetRequest) | [SendPasswordResetResponse](#moth-admin-v1-SendPasswordResetResponse) | SendPasswordReset emails the user a password-reset link, as if they had used &#34;forgot password&#34; themselves. |

 



<a name="moth_auth_v1_auth-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/auth/v1/auth.proto



<a name="moth-auth-v1-ChangePasswordRequest"></a>

### ChangePasswordRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| current_password | [string](#string) |  |  |
| new_password | [string](#string) |  |  |






<a name="moth-auth-v1-ChangePasswordResponse"></a>

### ChangePasswordResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tokens | [TokenPair](#moth-auth-v1-TokenPair) |  | A fresh session for this device; all other sessions are revoked. |






<a name="moth-auth-v1-ConfirmEmailChangeRequest"></a>

### ConfirmEmailChangeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |






<a name="moth-auth-v1-ConfirmEmailChangeResponse"></a>

### ConfirmEmailChangeResponse







<a name="moth-auth-v1-ConfirmEmailVerificationRequest"></a>

### ConfirmEmailVerificationRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |






<a name="moth-auth-v1-ConfirmEmailVerificationResponse"></a>

### ConfirmEmailVerificationResponse







<a name="moth-auth-v1-ConfirmPasswordResetRequest"></a>

### ConfirmPasswordResetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |
| new_password | [string](#string) |  |  |






<a name="moth-auth-v1-ConfirmPasswordResetResponse"></a>

### ConfirmPasswordResetResponse







<a name="moth-auth-v1-DeleteAccountRequest"></a>

### DeleteAccountRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| password | [string](#string) |  | Fresh re-authentication: the current password. (Recent social sign-in for social-only users arrives with milestone 04.) |






<a name="moth-auth-v1-DeleteAccountResponse"></a>

### DeleteAccountResponse







<a name="moth-auth-v1-ExchangeOAuthCodeRequest"></a>

### ExchangeOAuthCodeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [string](#string) |  | The one-time code from the web-redirect callback. |
| device_info | [string](#string) |  |  |






<a name="moth-auth-v1-ExchangeOAuthCodeResponse"></a>

### ExchangeOAuthCodeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-auth-v1-User) |  |  |
| tokens | [TokenPair](#moth-auth-v1-TokenPair) |  |  |






<a name="moth-auth-v1-GetMeRequest"></a>

### GetMeRequest







<a name="moth-auth-v1-GetMeResponse"></a>

### GetMeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-auth-v1-User) |  |  |






<a name="moth-auth-v1-RefreshTokenRequest"></a>

### RefreshTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| refresh_token | [string](#string) |  |  |






<a name="moth-auth-v1-RefreshTokenResponse"></a>

### RefreshTokenResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-auth-v1-User) |  |  |
| tokens | [TokenPair](#moth-auth-v1-TokenPair) |  |  |






<a name="moth-auth-v1-RequestEmailChangeRequest"></a>

### RequestEmailChangeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| new_email | [string](#string) |  |  |






<a name="moth-auth-v1-RequestEmailChangeResponse"></a>

### RequestEmailChangeResponse







<a name="moth-auth-v1-RequestEmailVerificationRequest"></a>

### RequestEmailVerificationRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |






<a name="moth-auth-v1-RequestEmailVerificationResponse"></a>

### RequestEmailVerificationResponse







<a name="moth-auth-v1-RequestPasswordResetRequest"></a>

### RequestPasswordResetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |






<a name="moth-auth-v1-RequestPasswordResetResponse"></a>

### RequestPasswordResetResponse







<a name="moth-auth-v1-SignInRequest"></a>

### SignInRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |
| password | [string](#string) |  |  |
| device_info | [string](#string) |  |  |






<a name="moth-auth-v1-SignInResponse"></a>

### SignInResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-auth-v1-User) |  |  |
| tokens | [TokenPair](#moth-auth-v1-TokenPair) |  |  |






<a name="moth-auth-v1-SignInWithOAuthRequest"></a>

### SignInWithOAuthRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [OAuthProvider](#moth-auth-v1-OAuthProvider) |  |  |
| id_token | [string](#string) |  | The provider-issued OIDC ID token (JWT). |
| nonce | [string](#string) |  | The raw per-attempt nonce the SDK generated for this sign-in. The server requires the ID token&#39;s `nonce` claim to match (Apple carries its SHA-256 per their scheme), so replayed ID tokens are rejected. |
| authorization_code | [string](#string) |  | Apple only: the authorization code from the native flow, exchanged server-side for the refresh token that account deletion later revokes (App Store requirement). |
| given_name | [string](#string) |  | Apple only: the user&#39;s name, which Apple exposes solely to the app and solely on first authorization. Client-asserted — used for the initial display name, never for identity resolution. |
| family_name | [string](#string) |  |  |
| device_info | [string](#string) |  | Free-form device description stored with the session, e.g. &#34;iPhone 15&#34;. |






<a name="moth-auth-v1-SignInWithOAuthResponse"></a>

### SignInWithOAuthResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-auth-v1-User) |  |  |
| tokens | [TokenPair](#moth-auth-v1-TokenPair) |  |  |






<a name="moth-auth-v1-SignOutRequest"></a>

### SignOutRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| refresh_token | [string](#string) |  |  |
| all_devices | [bool](#bool) |  | Revoke every session of the user, not just this one. |






<a name="moth-auth-v1-SignOutResponse"></a>

### SignOutResponse







<a name="moth-auth-v1-SignUpRequest"></a>

### SignUpRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |
| password | [string](#string) |  |  |
| display_name | [string](#string) |  |  |
| device_info | [string](#string) |  | Free-form device description stored with the session, e.g. &#34;iPhone 15&#34;. |
| captcha_token | [string](#string) |  | Optional CAPTCHA solution, verified when the project configures a captcha_verify_url (off by default; enforcement is a documented hook). |






<a name="moth-auth-v1-SignUpResponse"></a>

### SignUpResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-auth-v1-User) |  | Unset when project policy withholds it: enumeration-safe projects always return an empty response, and projects requiring verification return the user without tokens. |
| tokens | [TokenPair](#moth-auth-v1-TokenPair) |  | Set only when the user may sign in immediately. |






<a name="moth-auth-v1-TokenPair"></a>

### TokenPair
TokenPair is one authenticated session: a short-lived ES256 JWT plus the
opaque rotating refresh token that renews it.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_token | [string](#string) |  |  |
| refresh_token | [string](#string) |  |  |
| expires_in | [int64](#int64) |  | Access token lifetime in seconds. |






<a name="moth-auth-v1-UnlinkIdentityRequest"></a>

### UnlinkIdentityRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider | [OAuthProvider](#moth-auth-v1-OAuthProvider) |  |  |






<a name="moth-auth-v1-UnlinkIdentityResponse"></a>

### UnlinkIdentityResponse







<a name="moth-auth-v1-UpdateMeRequest"></a>

### UpdateMeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| display_name | [string](#string) | optional |  |
| avatar_url | [string](#string) | optional |  |






<a name="moth-auth-v1-UpdateMeResponse"></a>

### UpdateMeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-auth-v1-User) |  |  |






<a name="moth-auth-v1-User"></a>

### User
User is the caller&#39;s own account as exposed to the app.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| email | [string](#string) |  |  |
| email_verified | [bool](#bool) |  |  |
| display_name | [string](#string) |  |  |
| avatar_url | [string](#string) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |





 


<a name="moth-auth-v1-OAuthProvider"></a>

### OAuthProvider
OAuthProvider identifies a supported social sign-in provider.
(buf splits &#34;OAuth&#34; as &#34;O_Auth&#34;; the natural OAUTH_ prefix is kept.)

| Name | Number | Description |
| ---- | ------ | ----------- |
| OAUTH_PROVIDER_UNSPECIFIED | 0 | buf:lint:ignore ENUM_VALUE_PREFIX |
| OAUTH_PROVIDER_GOOGLE | 1 | buf:lint:ignore ENUM_VALUE_PREFIX |
| OAUTH_PROVIDER_APPLE | 2 | buf:lint:ignore ENUM_VALUE_PREFIX |


 

 


<a name="moth-auth-v1-AuthService"></a>

### AuthService
AuthService is the public end-user authentication API consumed by mobile
apps (via the SDK). Every call carries the project&#39;s publishable key in
`x-moth-key: pk_...` request metadata; an interceptor resolves it to the
project, so users, tokens and emails are always project-scoped.

RPCs about the current user (GetMe, UpdateMe, ChangePassword,
RequestEmailChange, DeleteAccount) additionally require a valid access
token in `authorization: Bearer ...` metadata.

Errors carry a google.rpc.ErrorInfo detail with a stable machine-readable
`reason` (e.g. INVALID_CREDENTIALS, EMAIL_NOT_VERIFIED) that SDKs map to
typed errors.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SignUp | [SignUpRequest](#moth-auth-v1-SignUpRequest) | [SignUpResponse](#moth-auth-v1-SignUpResponse) | SignUp registers a new email/password user, subject to project policy (public signup open, password length, email verification). Depending on policy the response may already include tokens, or be empty until the email is verified. |
| SignIn | [SignInRequest](#moth-auth-v1-SignInRequest) | [SignInResponse](#moth-auth-v1-SignInResponse) | SignIn exchanges email/password for a token pair. The error is the same whether the email is unknown or the password wrong. |
| RefreshToken | [RefreshTokenRequest](#moth-auth-v1-RefreshTokenRequest) | [RefreshTokenResponse](#moth-auth-v1-RefreshTokenResponse) | RefreshToken rotates the presented refresh token and mints a fresh access token. Presenting an already-rotated token is treated as theft: the whole token family is revoked. |
| SignOut | [SignOutRequest](#moth-auth-v1-SignOutRequest) | [SignOutResponse](#moth-auth-v1-SignOutResponse) | SignOut revokes the presented refresh token, or every session of the user with all_devices. |
| GetMe | [GetMeRequest](#moth-auth-v1-GetMeRequest) | [GetMeResponse](#moth-auth-v1-GetMeResponse) | GetMe returns the user authenticated by the access token. |
| UpdateMe | [UpdateMeRequest](#moth-auth-v1-UpdateMeRequest) | [UpdateMeResponse](#moth-auth-v1-UpdateMeResponse) | UpdateMe updates the user&#39;s own profile fields. |
| ChangePassword | [ChangePasswordRequest](#moth-auth-v1-ChangePasswordRequest) | [ChangePasswordResponse](#moth-auth-v1-ChangePasswordResponse) | ChangePassword requires the current password, revokes every other session and returns a fresh token pair for this device. |
| RequestEmailVerification | [RequestEmailVerificationRequest](#moth-auth-v1-RequestEmailVerificationRequest) | [RequestEmailVerificationResponse](#moth-auth-v1-RequestEmailVerificationResponse) | RequestEmailVerification (re)sends the verification email. It always returns OK so responses never reveal whether an account exists. |
| ConfirmEmailVerification | [ConfirmEmailVerificationRequest](#moth-auth-v1-ConfirmEmailVerificationRequest) | [ConfirmEmailVerificationResponse](#moth-auth-v1-ConfirmEmailVerificationResponse) | ConfirmEmailVerification consumes a verification token from the email link and marks the address verified. |
| RequestPasswordReset | [RequestPasswordResetRequest](#moth-auth-v1-RequestPasswordResetRequest) | [RequestPasswordResetResponse](#moth-auth-v1-RequestPasswordResetResponse) | RequestPasswordReset emails a reset link. It always returns OK so responses never reveal whether an account exists. |
| ConfirmPasswordReset | [ConfirmPasswordResetRequest](#moth-auth-v1-ConfirmPasswordResetRequest) | [ConfirmPasswordResetResponse](#moth-auth-v1-ConfirmPasswordResetResponse) | ConfirmPasswordReset consumes a reset token and sets the new password; every refresh token of the user is revoked. |
| RequestEmailChange | [RequestEmailChangeRequest](#moth-auth-v1-RequestEmailChangeRequest) | [RequestEmailChangeResponse](#moth-auth-v1-RequestEmailChangeResponse) | RequestEmailChange sends a confirmation link to the new address; the account email only switches once that address is verified. |
| ConfirmEmailChange | [ConfirmEmailChangeRequest](#moth-auth-v1-ConfirmEmailChangeRequest) | [ConfirmEmailChangeResponse](#moth-auth-v1-ConfirmEmailChangeResponse) | ConfirmEmailChange consumes an email-change token and applies the pending address. The previous address receives a notification with a revert link (valid 72 h) that goes through this same RPC. |
| SignInWithOAuth | [SignInWithOAuthRequest](#moth-auth-v1-SignInWithOAuthRequest) | [SignInWithOAuthResponse](#moth-auth-v1-SignInWithOAuthResponse) | SignInWithOAuth signs in (or up) with a provider ID token obtained by a native Google/Apple flow on the device. The token is verified server-side (signature against the provider JWKS, issuer, audience against the project&#39;s configured client/bundle IDs, expiry, nonce); email, name and subject only ever come from the verified token. Account resolution: an existing (provider, subject) identity signs that user in; else a provider-verified email matching an existing user links a new identity to it (when the project&#39;s auto_link_verified_email policy allows); else a new user is created. |
| ExchangeOAuthCode | [ExchangeOAuthCodeRequest](#moth-auth-v1-ExchangeOAuthCodeRequest) | [ExchangeOAuthCodeResponse](#moth-auth-v1-ExchangeOAuthCodeResponse) | ExchangeOAuthCode trades the one-time code minted by the web-redirect fallback flow (GET /oauth/{provider}/start → provider consent → callback → redirect back into the app) for a token pair. Codes are single-use and short-lived. |
| UnlinkIdentity | [UnlinkIdentityRequest](#moth-auth-v1-UnlinkIdentityRequest) | [UnlinkIdentityResponse](#moth-auth-v1-UnlinkIdentityResponse) | UnlinkIdentity removes the caller&#39;s identity for one provider. Requires a Bearer access token. Refused when it would leave the account without any way to sign in. |
| DeleteAccount | [DeleteAccountRequest](#moth-auth-v1-DeleteAccountRequest) | [DeleteAccountResponse](#moth-auth-v1-DeleteAccountResponse) | DeleteAccount permanently deletes the user after fresh re-authentication (App Store guideline 5.1.1). Identities, sessions and email tokens are cascaded. |

 



<a name="moth_auth_v1_config-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/auth/v1/config.proto



<a name="moth-auth-v1-AppleConfig"></a>

### AppleConfig
AppleConfig is the public part of a project&#39;s Sign in with Apple
configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |






<a name="moth-auth-v1-Copy"></a>

### Copy
Copy is the resolved, localized copy for the negotiated locale: the message
key → localized-string map the SDK renders its auth screens from
(sign_in.*, sign_up.*, password_reset.*, verify_email.*), already merged
bundled-default → project-override. The locale is negotiated server-side
from the request&#39;s Accept-Language / x-moth-language metadata against the
project&#39;s available locales; the client never dictates raw copy.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| copy_revision | [string](#string) |  | Opaque cache token identifying this (locale, override-revision) pair. It changes whenever the negotiated locale or the project&#39;s copy overrides change. Cache `messages` keyed by this value and echo it as GetProjectConfigRequest.known_copy_revision; the response omits `messages` when it still matches (see the caching contract on the request). |
| locale | [string](#string) |  | The negotiated BCP-47 locale this copy is for (e.g. &#34;fr&#34;). Echoed so the client sets lang/dir correctly and re-requests when the device language changes; always present even when `messages` is omitted. |
| messages | [Copy.MessagesEntry](#moth-auth-v1-Copy-MessagesEntry) | repeated | Resolved message key → localized string for the negotiated locale. |






<a name="moth-auth-v1-Copy-MessagesEntry"></a>

### Copy.MessagesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="moth-auth-v1-GetProjectConfigRequest"></a>

### GetProjectConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| known_theme_revision | [string](#string) |  | Theme caching contract: pass the revision_id of the theme the client has cached (empty on first call). When it still matches the current revision, the response omits `theme` entirely — the client keeps rendering its cached copy. When it differs (or was empty), `theme` is present and the client replaces its cache. |
| known_copy_revision | [string](#string) |  | Copy caching contract (identical shape to the theme one, but keyed by the negotiated locale too): pass the copy_revision the client has cached for the locale it is about to render (empty on first call). When it still matches the token the server computes for the negotiated locale, the response&#39;s `copy` carries the locale &#43; copy_revision but omits `messages` (stale-while-revalidate); when it differs (or was empty), `messages` is present and the client replaces its cache. The negotiated locale comes from Accept-Language / x-moth-language metadata, never from this body. |






<a name="moth-auth-v1-GetProjectConfigResponse"></a>

### GetProjectConfigResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| google | [GoogleConfig](#moth-auth-v1-GoogleConfig) |  |  |
| apple | [AppleConfig](#moth-auth-v1-AppleConfig) |  |  |
| password_min_length | [int32](#int32) |  | Minimum accepted password length. |
| sign_up_open | [bool](#bool) |  | Whether the public SignUp RPC is open. |
| theme | [Theme](#moth-auth-v1-Theme) |  | The project&#39;s design system. Omitted when GetProjectConfigRequest.known_theme_revision matches the current revision (see the caching contract there); always present otherwise, including for projects on the built-in default theme. |
| copy | [Copy](#moth-auth-v1-Copy) |  | The localized copy for the negotiated locale. Always present (it carries the negotiated locale &#43; copy_revision so the client caches per (locale, revision)); its `messages` map is omitted when GetProjectConfigRequest.known_copy_revision matches, present otherwise — including for projects with no copy overrides (fully bundled defaults). |






<a name="moth-auth-v1-GoogleConfig"></a>

### GoogleConfig
GoogleConfig is the public part of a project&#39;s Sign in with Google
configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| web_client_id | [string](#string) |  | OAuth client IDs the native flows initialize with. Client IDs are public values (the secret never leaves the server). |
| ios_client_id | [string](#string) |  |  |
| android_client_id | [string](#string) |  |  |






<a name="moth-auth-v1-Theme"></a>

### Theme
Theme is the public, fully resolved form of the project&#39;s design system,
ready to render: dark colors are already derived server-side, asset
references are absolute URLs. Binary assets (logo images, font files)
stay plain-HTTP downloads with cache headers — they don&#39;t belong in RPC
responses.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revision_id | [string](#string) |  | Identifies this version of the theme; changes on every admin edit. Cache the theme keyed by this value and echo it as GetProjectConfigRequest.known_theme_revision. |
| colors | [ThemeColors](#moth-auth-v1-ThemeColors) |  | Light palette, &#34;#RRGGBB&#34; values. |
| dark_colors | [ThemeColors](#moth-auth-v1-ThemeColors) |  | Dark palette, fully resolved (admin overrides merged with derived values); render it when the device is in dark mode. |
| font_family | [string](#string) |  | Font family name (from the server&#39;s curated set). |
| font_url | [string](#string) |  | Absolute URL of the font file to download and register; cacheable. |
| font_scale | [double](#double) |  | Global text-size multiplier. |
| spacing_unit | [int32](#int32) |  | Base spacing step in logical pixels. |
| corner_radius | [int32](#int32) |  | Component corner radius in logical pixels. |
| logo_light_url | [string](#string) |  | Absolute logo URLs per color scheme; empty when no logo is set. |
| logo_dark_url | [string](#string) |  |  |
| terms_url | [string](#string) |  | Optional legal links rendered in the login screen footer. |
| privacy_url | [string](#string) |  |  |






<a name="moth-auth-v1-ThemeColors"></a>

### ThemeColors
ThemeColors is a complete palette: each color role and its &#34;on&#34;
(foreground) counterpart. Server-side validation guarantees WCAG AA
contrast (&gt;= 4.5:1) between every pair.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| primary | [string](#string) |  |  |
| on_primary | [string](#string) |  |  |
| background | [string](#string) |  |  |
| on_background | [string](#string) |  |  |
| surface | [string](#string) |  |  |
| on_surface | [string](#string) |  |  |
| error | [string](#string) |  |  |
| on_error | [string](#string) |  |  |





 

 

 


<a name="moth-auth-v1-ConfigService"></a>

### ConfigService
ConfigService exposes a project&#39;s public, non-secret configuration to the
mobile SDK, so the login screen can render exactly the sign-in methods
the project enables. Authenticated like AuthService: every call carries
the project&#39;s publishable key in `x-moth-key: pk_...` request metadata.

Later milestones extend GetProjectConfigResponse: SDK bootstrap values in
05, login-screen branding/theme in 06. Fields are only ever added.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetProjectConfig | [GetProjectConfigRequest](#moth-auth-v1-GetProjectConfigRequest) | [GetProjectConfigResponse](#moth-auth-v1-GetProjectConfigResponse) | GetProjectConfig returns the project configuration a client may see. Never includes secrets; only values that are safe to embed in an app. |

 



<a name="moth_billing_v1_billing-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/billing/v1/billing.proto



<a name="moth-billing-v1-ActiveSubscription"></a>

### ActiveSubscription
ActiveSubscription is one of the user&#39;s store subscriptions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| product_identifier | [string](#string) |  | The moth product identifier, when the store SKU is mapped; empty otherwise. |
| store | [Store](#moth-billing-v1-Store) |  |  |
| status | [SubscriptionStatus](#moth-billing-v1-SubscriptionStatus) |  |  |
| current_period_end | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | End of the current paid (or trial) period; the renewal date when auto_renew is true. |
| auto_renew | [bool](#bool) |  |  |
| is_sandbox | [bool](#bool) |  | Whether this subscription is a sandbox/test purchase. |






<a name="moth-billing-v1-Copy"></a>

### Copy
Copy is the resolved, localized paywall copy for the negotiated locale: the
paywall.* message key → localized-string map (headline, subtitle, benefit
bullets, CTA, legal labels), merged bundled-default → project-override. The
paywall copy keys are part of the same catalog as the auth-screen copy
(moth.auth.v1). The locale is negotiated server-side from the request&#39;s
Accept-Language / x-moth-language metadata; the client never dictates raw
copy. The structural Paywall message above stays authoritative for
layout/offering/tier selection.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| copy_revision | [string](#string) |  | Opaque cache token identifying this (locale, override-revision) pair. Cache `messages` keyed by it and echo it as GetPaywallRequest.known_copy_revision; the response omits `messages` when it still matches. |
| locale | [string](#string) |  | The negotiated BCP-47 locale this copy is for (e.g. &#34;fr&#34;). |
| messages | [Copy.MessagesEntry](#moth-billing-v1-Copy-MessagesEntry) | repeated | Resolved paywall.* message key → localized string for the negotiated locale. |






<a name="moth-billing-v1-Copy-MessagesEntry"></a>

### Copy.MessagesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="moth-billing-v1-CreateBillingPortalSessionRequest"></a>

### CreateBillingPortalSessionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| return_url | [string](#string) |  | Where Stripe redirects the browser when the user leaves the portal. |






<a name="moth-billing-v1-CreateBillingPortalSessionResponse"></a>

### CreateBillingPortalSessionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  | The Stripe-hosted Billing Portal URL to redirect the browser to. |






<a name="moth-billing-v1-CreateCheckoutSessionRequest"></a>

### CreateCheckoutSessionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| product_identifier | [string](#string) |  | The moth product identifier to subscribe to (its own catalog id, not the Stripe price id). The tier must carry a stripe_price_id. |
| success_url | [string](#string) |  | Where Stripe redirects the browser after a completed checkout. |
| cancel_url | [string](#string) |  | Where Stripe redirects the browser when the user backs out. |






<a name="moth-billing-v1-CreateCheckoutSessionResponse"></a>

### CreateCheckoutSessionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  | The Stripe-hosted Checkout URL to redirect the browser to. |






<a name="moth-billing-v1-CustomerInfo"></a>

### CustomerInfo
CustomerInfo is the complete subscription picture for one user. Apps gate
features on active_entitlements (never on a product id): check whether the
stable entitlement identifier (e.g. &#34;pro&#34;) is present.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| active_entitlements | [Entitlement](#moth-billing-v1-Entitlement) | repeated | The entitlements the user currently holds. Empty means the free `none` tier — a valid, expected state, not an error. |
| subscriptions | [ActiveSubscription](#moth-billing-v1-ActiveSubscription) | repeated | The user&#39;s known subscriptions across stores (may include inactive ones, for history/paywall display). |






<a name="moth-billing-v1-Entitlement"></a>

### Entitlement
Entitlement is one active capability the user holds.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [string](#string) |  | Stable identifier the app checks (e.g. &#34;pro&#34;). Never changes across app releases even when the granting product does. |
| expire_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the entitlement lapses; unset for a non-expiring grant. |
| source | [EntitlementSource](#moth-billing-v1-EntitlementSource) |  | Why it is active (store subscription vs operator grant). |
| product_identifier | [string](#string) |  | The moth product identifier that granted it, when source is STORE; empty for grants. |






<a name="moth-billing-v1-GetCustomerInfoRequest"></a>

### GetCustomerInfoRequest







<a name="moth-billing-v1-GetCustomerInfoResponse"></a>

### GetCustomerInfoResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| customer_info | [CustomerInfo](#moth-billing-v1-CustomerInfo) |  |  |






<a name="moth-billing-v1-GetOfferingsRequest"></a>

### GetOfferingsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| offering | [string](#string) |  | Offering tag; empty selects the project&#39;s default offering. |






<a name="moth-billing-v1-GetOfferingsResponse"></a>

### GetOfferingsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| offering | [Offering](#moth-billing-v1-Offering) |  |  |






<a name="moth-billing-v1-GetPaywallRequest"></a>

### GetPaywallRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| known_paywall_revision | [string](#string) |  | The revision_id of the paywall the client has cached (empty on first call). When it still matches the current revision, the response omits `paywall`; see the caching contract on GetPaywall. |
| known_copy_revision | [string](#string) |  | The copy_revision the client has cached for the locale it is about to render (empty on first call). When it still matches the token the server computes for the negotiated locale, the response&#39;s `copy` carries the locale &#43; copy_revision but omits `messages`; when it differs (or was empty), `messages` is present. The negotiated locale comes from Accept-Language / x-moth-language metadata, never from this body. |






<a name="moth-billing-v1-GetPaywallResponse"></a>

### GetPaywallResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| paywall | [Paywall](#moth-billing-v1-Paywall) |  | Omitted when GetPaywallRequest.known_paywall_revision matches the current revision; present otherwise (including for projects on the built-in default paywall config). |
| copy | [Copy](#moth-billing-v1-Copy) |  | The localized paywall copy for the negotiated locale. Always present (it carries the negotiated locale &#43; copy_revision); its `messages` map is omitted when GetPaywallRequest.known_copy_revision matches, present otherwise — including for projects with no copy overrides. |






<a name="moth-billing-v1-Offering"></a>

### Offering
Offering is the ordered set of products a paywall presents — the products
sharing an `offering` tag, in sort order. Every project has a default
offering (&#34;default&#34;).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [string](#string) |  | Offering tag; &#34;default&#34; for the project&#39;s default offering. |
| is_default | [bool](#bool) |  |  |
| products | [OfferingProduct](#moth-billing-v1-OfferingProduct) | repeated | The products to display, in paywall order. |






<a name="moth-billing-v1-OfferingProduct"></a>

### OfferingProduct
OfferingProduct is one purchasable tier as the paywall needs it: enough to
render a card and match the native store product. Price/period are display
&#43; analytics metadata; the native store read stays authoritative for the
localized price actually charged.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [string](#string) |  | Stable moth catalog identifier (e.g. &#34;monthly&#34;); the app never gates on this — it gates on entitlements — but the SDK uses it to drive purchases. |
| display_name | [string](#string) |  |  |
| apple_product_id | [string](#string) |  | Store SKUs so the SDK can pair this tier with the native store product; either may be empty when the tier ships on one store only. |
| google_product_id | [string](#string) |  |  |
| billing_period | [string](#string) |  |  |
| price_amount_micros | [int64](#int64) |  |  |
| currency | [string](#string) |  |  |
| trial_period | [string](#string) |  | Trial/intro descriptor (display &#43; analytics only). |
| intro_price_amount_micros | [int64](#int64) |  |  |
| intro_period | [string](#string) |  |  |
| entitlements | [string](#string) | repeated | The stable entitlement identifiers this product grants while active (e.g. &#34;pro&#34;), so the paywall can label what a tier unlocks. |
| sort_order | [int32](#int32) |  |  |
| highlighted | [bool](#bool) |  | Whether this tier is the paywall&#39;s highlighted &#34;most popular&#34; tier (from the paywall config&#39;s highlighted_product_identifier). |
| stripe_price_id | [string](#string) |  | Stripe recurring Price id (&#34;price_...&#34;); empty when the tier does not sell on the web (the React paywall marks such tiers unavailable). |






<a name="moth-billing-v1-Paywall"></a>

### Paywall
Paywall is the public, render-ready paywall configuration. Copy and layout
only — colors/typography inherit from the theme.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revision_id | [string](#string) |  | Identifies this version of the paywall config; changes on every admin edit. Cache the paywall keyed by this value and echo it as GetPaywallRequest.known_paywall_revision. |
| headline | [string](#string) |  |  |
| subtitle | [string](#string) |  |  |
| benefits | [string](#string) | repeated | Feature/benefit bullets, in display order. |
| offering | [string](#string) |  | The offering tag whose products this paywall lists; pass it to GetOfferings.offering. Empty selects the default offering. |
| highlighted_product_identifier | [string](#string) |  | The product identifier to render as &#34;most popular&#34;; empty for none. |
| layout | [PaywallLayout](#moth-billing-v1-PaywallLayout) |  |  |
| terms_url | [string](#string) |  | Optional legal links rendered in the paywall footer. |
| privacy_url | [string](#string) |  |  |






<a name="moth-billing-v1-RestorePurchasesRequest"></a>

### RestorePurchasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| store | [Store](#moth-billing-v1-Store) |  |  |
| receipts | [string](#string) | repeated | The receipts to re-link. For Apple, StoreKit 2 signed transactions (JWS); for Google, Play Billing purchase tokens. |






<a name="moth-billing-v1-RestorePurchasesResponse"></a>

### RestorePurchasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| customer_info | [CustomerInfo](#moth-billing-v1-CustomerInfo) |  |  |






<a name="moth-billing-v1-SubmitPurchaseRequest"></a>

### SubmitPurchaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| store | [Store](#moth-billing-v1-Store) |  |  |
| product_identifier | [string](#string) |  | The moth product identifier the app is purchasing (its own catalog id, not the store SKU). moth maps it to the store product for validation. |
| apple_jws_transaction | [string](#string) |  |  |
| google_purchase_token | [string](#string) |  |  |
| google_subscription_id | [string](#string) |  | Google Play subscription id (the store product id); required alongside google_purchase_token, ignored for Apple. |






<a name="moth-billing-v1-SubmitPurchaseResponse"></a>

### SubmitPurchaseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| customer_info | [CustomerInfo](#moth-billing-v1-CustomerInfo) |  |  |





 


<a name="moth-billing-v1-EntitlementSource"></a>

### EntitlementSource
EntitlementSource explains why an entitlement is active.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENTITLEMENT_SOURCE_UNSPECIFIED | 0 |  |
| ENTITLEMENT_SOURCE_STORE | 1 | Granted by an active store subscription. |
| ENTITLEMENT_SOURCE_GRANT | 2 | Granted by an operator (promo/comp), independent of store state. |
| ENTITLEMENT_SOURCE_NONE | 3 | The built-in free tier (no active subscription or grant). Reserved; the free tier is normally conveyed by an empty active_entitlements list. |



<a name="moth-billing-v1-PaywallLayout"></a>

### PaywallLayout
PaywallLayout is the rendering variant the paywall screen uses; the token
space (colors/spacing/radius) always comes from the theme.

| Name | Number | Description |
| ---- | ------ | ----------- |
| PAYWALL_LAYOUT_UNSPECIFIED | 0 |  |
| PAYWALL_LAYOUT_TILES | 1 | One card per tier, side by side (the default). |
| PAYWALL_LAYOUT_LIST | 2 | Tiers stacked as full-width rows. |
| PAYWALL_LAYOUT_COMPACT | 3 | A single selected tier with a period toggle. |



<a name="moth-billing-v1-Store"></a>

### Store
Store identifies which app store a purchase or subscription belongs to.

| Name | Number | Description |
| ---- | ------ | ----------- |
| STORE_UNSPECIFIED | 0 |  |
| STORE_APPLE | 1 |  |
| STORE_GOOGLE | 2 |  |
| STORE_STRIPE | 3 |  |



<a name="moth-billing-v1-SubscriptionStatus"></a>

### SubscriptionStatus
SubscriptionStatus mirrors the store&#39;s renewal state, mapped to a small set
common to Apple and Google. active/trialing/in_grace_period/in_billing_retry
all keep access; paused/expired/revoked do not.

| Name | Number | Description |
| ---- | ------ | ----------- |
| SUBSCRIPTION_STATUS_UNSPECIFIED | 0 |  |
| SUBSCRIPTION_STATUS_ACTIVE | 1 |  |
| SUBSCRIPTION_STATUS_TRIALING | 2 |  |
| SUBSCRIPTION_STATUS_IN_GRACE_PERIOD | 3 |  |
| SUBSCRIPTION_STATUS_IN_BILLING_RETRY | 4 | Google &#34;on hold&#34;: the renewal is being retried after a payment failure. |
| SUBSCRIPTION_STATUS_PAUSED | 5 |  |
| SUBSCRIPTION_STATUS_EXPIRED | 6 |  |
| SUBSCRIPTION_STATUS_REVOKED | 7 |  |


 

 


<a name="moth-billing-v1-BillingService"></a>

### BillingService
BillingService is the client-facing subscription API, consumed by the moth
Flutter SDK. Authenticated exactly like AuthService: every call carries the
project publishable key in `x-moth-key: pk_...` request metadata AND a user
access token in the `Authorization: Bearer &lt;jwt&gt;` header — every RPC is
scoped to the signed-in user.

The core contract: **a user always has a valid subscription state.** A
never-paid user, a free-tier user, and a user in a project that has declared
no products all get a well-formed CustomerInfo with an empty
active_entitlements list (the built-in `none` tier) — never an error. moth is
a validating mirror of the store: it never marks a subscription active on the
client&#39;s say-so, only after verifying a signed transaction or reading the
store&#39;s authoritative state.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetCustomerInfo | [GetCustomerInfoRequest](#moth-billing-v1-GetCustomerInfoRequest) | [GetCustomerInfoResponse](#moth-billing-v1-GetCustomerInfoResponse) | GetCustomerInfo returns the signed-in user&#39;s active entitlements and subscriptions. Always succeeds with a valid object; `none` (empty entitlements) for free users. Cheap and safe to call on every app launch. |
| SubmitPurchase | [SubmitPurchaseRequest](#moth-billing-v1-SubmitPurchaseRequest) | [SubmitPurchaseResponse](#moth-billing-v1-SubmitPurchaseResponse) | SubmitPurchase hands moth the receipt of a purchase the app just completed natively. moth validates it against the store, links the subscription to the current user, derives entitlements, and returns the fresh CustomerInfo. |
| RestorePurchases | [RestorePurchasesRequest](#moth-billing-v1-RestorePurchasesRequest) | [RestorePurchasesResponse](#moth-billing-v1-RestorePurchasesResponse) | RestorePurchases re-links a user&#39;s existing store purchases to the current account (new device, reinstall, account change), applying the store&#39;s own transfer rules, then returns the fresh CustomerInfo. |
| GetOfferings | [GetOfferingsRequest](#moth-billing-v1-GetOfferingsRequest) | [GetOfferingsResponse](#moth-billing-v1-GetOfferingsResponse) | GetOfferings returns an offering&#39;s products for the paywall to display: per product the catalog identifier, display name, store SKUs (so the SDK can match the native store products), price/period metadata, trial/intro descriptor, the entitlements it grants, sort order and the &#34;most popular&#34; highlight flag. Unlike the three RPCs above this is publishable-key only (no Bearer): a paywall renders before the user signs in. |
| GetPaywall | [GetPaywallRequest](#moth-billing-v1-GetPaywallRequest) | [GetPaywallResponse](#moth-billing-v1-GetPaywallResponse) | GetPaywall returns the project&#39;s public paywall configuration (copy, benefit bullets, offering ref, layout, highlighted tier, legal links) with a revision id, for the SDK&#39;s batteries-included paywall screen. Colors and typography are NOT here — the paywall inherits them from the theme (GetProjectConfig, milestone 06).

Caching contract (identical to GetProjectConfig &#43; theme): the client caches the paywall keyed by revision_id and echoes it as GetPaywallRequest.known_paywall_revision. When it still matches, the response omits `paywall` and the client keeps rendering its cache (stale-while-revalidate); when it differs (or was empty on first call), `paywall` is present and the client replaces its cache. Publishable-key only, like GetOfferings. |
| CreateCheckoutSession | [CreateCheckoutSessionRequest](#moth-billing-v1-CreateCheckoutSessionRequest) | [CreateCheckoutSessionResponse](#moth-billing-v1-CreateCheckoutSessionResponse) | CreateCheckoutSession starts a Stripe-hosted Checkout for a subscription to the tier&#39;s Stripe price, bound to the signed-in user&#39;s Stripe customer (created on demand). moth never renders a card field: the response is a redirect URL to Stripe&#39;s hosted Checkout, and the resulting subscription lands through the webhook like any other store event. Requires Bearer (like GetCustomerInfo); fails with a precondition error when the project has no Stripe credentials, and an invalid-argument error when the tier has no Stripe price. |
| CreateBillingPortalSession | [CreateBillingPortalSessionRequest](#moth-billing-v1-CreateBillingPortalSessionRequest) | [CreateBillingPortalSessionResponse](#moth-billing-v1-CreateBillingPortalSessionResponse) | CreateBillingPortalSession returns a Stripe Billing Portal URL for the signed-in user — cancel, payment-method and invoice management stay Stripe-hosted, the web analogue of deep-linking to the stores&#39; subscription-management UI. Requires Bearer. |

 



<a name="moth_projectconfig_v1_projectconfig-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/projectconfig/v1/projectconfig.proto



<a name="moth-projectconfig-v1-CacheEnvelope"></a>

### CacheEnvelope
CacheEnvelope wraps a config payload the Flutter SDK persists on device
(theme, paywall, copy — milestone 16 caches, re-encoded from JSON to
protobuf). payload is the serialized wire message exactly as the server
delivered it (moth.auth.v1.Theme / moth.billing.v1.Paywall /
moth.auth.v1.Copy), so the cache and the wire share one schema. The SDK
serves the cached payload without any network call until
fetched_at_unix_ms &#43; its configured TTL has passed, then revalidates
cheaply with the known_*_revision request fields.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [bytes](#bytes) |  |  |
| revision | [string](#string) |  | revision is the server revision the payload came from (theme/paywall/copy revision id) — the revalidation key. |
| locale | [string](#string) |  | locale is the negotiated BCP-47 tag for locale-keyed payloads (copy); empty for locale-independent payloads. |
| fetched_at_unix_ms | [int64](#int64) |  | fetched_at_unix_ms is when the payload was fetched or last revalidated, Unix milliseconds UTC. |






<a name="moth-projectconfig-v1-CopyLocaleMessages"></a>

### CopyLocaleMessages
CopyLocaleMessages is one locale&#39;s copy overrides: catalog message key
(e.g. &#34;sign_in.title&#34;) to the operator-customized string.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| messages | [CopyLocaleMessages.MessagesEntry](#moth-projectconfig-v1-CopyLocaleMessages-MessagesEntry) | repeated |  |






<a name="moth-projectconfig-v1-CopyLocaleMessages-MessagesEntry"></a>

### CopyLocaleMessages.MessagesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="moth-projectconfig-v1-LegalLinks"></a>

### LegalLinks
LegalLinks are the optional legal URLs rendered near signup and on the
paywall footer.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| terms_url | [string](#string) |  |  |
| privacy_url | [string](#string) |  |  |






<a name="moth-projectconfig-v1-StoredCopy"></a>

### StoredCopy
StoredCopy is one project&#39;s localization overrides as persisted on the
project row and in copy_revisions (milestone 15, re-encoded from JSON to
protobuf): BCP-47 locale tag to that locale&#39;s key overrides. Bundled
catalog defaults live in the binary (internal/i18n), never here.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| locales | [StoredCopy.LocalesEntry](#moth-projectconfig-v1-StoredCopy-LocalesEntry) | repeated |  |






<a name="moth-projectconfig-v1-StoredCopy-LocalesEntry"></a>

### StoredCopy.LocalesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [CopyLocaleMessages](#moth-projectconfig-v1-CopyLocaleMessages) |  |  |






<a name="moth-projectconfig-v1-StoredPaywall"></a>

### StoredPaywall
StoredPaywall is one project&#39;s paywall configuration as persisted on the
project row and in paywall_revisions (milestone 13, re-encoded from JSON
to protobuf). Colors/typography always inherit from the theme — the
paywall owns no design tokens.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [int32](#int32) |  | version is the document schema version (internal/paywall.SchemaVersion). |
| headline | [string](#string) |  |  |
| subtitle | [string](#string) |  |  |
| benefits | [string](#string) | repeated |  |
| offering | [string](#string) |  | offering names the product offering the paywall presents; empty = the project&#39;s default offering. |
| highlighted_identifier | [string](#string) |  | highlighted_identifier marks the &#34;most popular&#34; tier; empty = none. |
| layout | [string](#string) |  |  |
| legal | [LegalLinks](#moth-projectconfig-v1-LegalLinks) |  |  |






<a name="moth-projectconfig-v1-StoredTheme"></a>

### StoredTheme
StoredTheme is one project&#39;s complete design system as persisted on the
project row and in theme_revisions (milestone 06, re-encoded from JSON to
protobuf). internal/theme owns validation and dark-palette derivation;
this message owns only the shape.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [int32](#int32) |  | version is the document schema version (internal/theme.SchemaVersion). |
| colors | [ThemeColors](#moth-projectconfig-v1-ThemeColors) |  |  |
| dark_colors | [ThemeColorOverrides](#moth-projectconfig-v1-ThemeColorOverrides) |  | dark_colors optionally overrides individual dark-palette colors; absent = fully derived from colors. |
| typography | [ThemeTypography](#moth-projectconfig-v1-ThemeTypography) |  |  |
| spacing | [ThemeSpacing](#moth-projectconfig-v1-ThemeSpacing) |  |  |
| shape | [ThemeShape](#moth-projectconfig-v1-ThemeShape) |  |  |
| logo | [ThemeLogo](#moth-projectconfig-v1-ThemeLogo) |  |  |
| legal | [LegalLinks](#moth-projectconfig-v1-LegalLinks) |  |  |






<a name="moth-projectconfig-v1-ThemeColorOverrides"></a>

### ThemeColorOverrides
ThemeColorOverrides is a partial dark palette: any empty field is derived
from the light palette instead (see internal/theme.DeriveDark).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| primary | [string](#string) |  |  |
| on_primary | [string](#string) |  |  |
| background | [string](#string) |  |  |
| on_background | [string](#string) |  |  |
| surface | [string](#string) |  |  |
| on_surface | [string](#string) |  |  |
| error | [string](#string) |  |  |
| on_error | [string](#string) |  |  |






<a name="moth-projectconfig-v1-ThemeColors"></a>

### ThemeColors
ThemeColors is a complete palette: every role and its &#34;on&#34; (foreground)
counterpart, as #RRGGBB values.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| primary | [string](#string) |  |  |
| on_primary | [string](#string) |  |  |
| background | [string](#string) |  |  |
| on_background | [string](#string) |  |  |
| surface | [string](#string) |  |  |
| on_surface | [string](#string) |  |  |
| error | [string](#string) |  |  |
| on_error | [string](#string) |  |  |






<a name="moth-projectconfig-v1-ThemeLogo"></a>

### ThemeLogo
ThemeLogo holds the server-managed asset paths of the uploaded logos,
one per color scheme (&#34;/assets/{project}/logo-light.png&#34;). Empty = none.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| light | [string](#string) |  |  |
| dark | [string](#string) |  |  |






<a name="moth-projectconfig-v1-ThemeShape"></a>

### ThemeShape
ThemeShape controls component rounding, in logical pixels.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| corner_radius | [int32](#int32) |  |  |






<a name="moth-projectconfig-v1-ThemeSpacing"></a>

### ThemeSpacing
ThemeSpacing is the base spacing grid step in logical pixels.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| unit | [int32](#int32) |  |  |






<a name="moth-projectconfig-v1-ThemeTypography"></a>

### ThemeTypography
ThemeTypography selects one of the curated embedded fonts and a global
size multiplier.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| font_family | [string](#string) |  |  |
| scale | [double](#double) |  |  |





 

 

 

 



<a name="moth_server_v1_entitlement-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/server/v1/entitlement.proto



<a name="moth-server-v1-Entitlement"></a>

### Entitlement
Entitlement is one active capability a user holds, derived from store
subscriptions and operator grants.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [string](#string) |  | Stable identifier the app gates on (e.g. &#34;pro&#34;). |
| expire_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the entitlement lapses; unset for a non-expiring grant. |
| source | [string](#string) |  | Why it is active: &#34;store&#34; (an active subscription) or &#34;grant&#34; (an operator promo/comp). |
| product_identifier | [string](#string) |  | The moth product identifier that granted it, when source is &#34;store&#34;. |
| is_sandbox | [bool](#bool) |  | True when this entitlement is backed by a store sandbox / license-tester subscription rather than a real production purchase. A backend gating paid features server-side should treat a sandbox entitlement as non-paying. Always false for operator grants. |






<a name="moth-server-v1-GetUserEntitlementsRequest"></a>

### GetUserEntitlementsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_id | [string](#string) |  |  |






<a name="moth-server-v1-GetUserEntitlementsResponse"></a>

### GetUserEntitlementsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| entitlements | [Entitlement](#moth-server-v1-Entitlement) | repeated |  |





 

 

 


<a name="moth-server-v1-EntitlementService"></a>

### EntitlementService
EntitlementService lets the developer&#39;s own backend read a user&#39;s derived
entitlements, so it can gate server-side features without trusting the
client. It returns exactly the set the client sees from
moth.billing.v1.GetCustomerInfo. Authenticated with the project secret key
(`x-moth-key: sk_...`); always scoped to that project.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetUserEntitlements | [GetUserEntitlementsRequest](#moth-server-v1-GetUserEntitlementsRequest) | [GetUserEntitlementsResponse](#moth-server-v1-GetUserEntitlementsResponse) | GetUserEntitlements returns a user&#39;s currently active entitlements. A user with no subscription and no grant returns an empty list (the free `none` tier) — not an error. |

 



<a name="moth_server_v1_token-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/server/v1/token.proto



<a name="moth-server-v1-IntrospectTokenRequest"></a>

### IntrospectTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_token | [string](#string) |  | The access token (JWT) exactly as presented by the client. |






<a name="moth-server-v1-IntrospectTokenResponse"></a>

### IntrospectTokenResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| active | [bool](#bool) |  | Whether the token is valid right now for this project. |
| inactive_reason | [string](#string) |  | Machine-readable cause when inactive: EXPIRED, INVALID_SIGNATURE, MALFORMED, USER_DISABLED, USER_NOT_FOUND. |
| user_id | [string](#string) |  | Claims, set only when active (except user_id which is set whenever the signature verified). |
| email | [string](#string) |  |  |
| email_verified | [bool](#bool) |  |  |
| custom_claims | [google.protobuf.Struct](#google-protobuf-Struct) |  | The user&#39;s custom claims as embedded in the token. |
| issue_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| expire_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |





 

 

 


<a name="moth-server-v1-TokenService"></a>

### TokenService
TokenService lets the developer&#39;s backend verify moth access tokens
online. Offline JWKS verification is the recommended default; use
introspection when instant revocation matters more than latency.

Every call carries the project secret key in `x-moth-key: sk_...`
request metadata (never shipped inside an app).

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| IntrospectToken | [IntrospectTokenRequest](#moth-server-v1-IntrospectTokenRequest) | [IntrospectTokenResponse](#moth-server-v1-IntrospectTokenResponse) | IntrospectToken reports whether an access token is currently valid for this project — including revocation and disabled-user state that offline JWT verification cannot see. |

 



<a name="moth_server_v1_user-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## moth/server/v1/user.proto



<a name="moth-server-v1-CreateUserRequest"></a>

### CreateUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |
| password | [string](#string) |  | Optional; empty creates a user that must reset before password sign-in. |
| display_name | [string](#string) |  |  |
| email_verified | [bool](#bool) |  |  |
| custom_claims | [google.protobuf.Struct](#google-protobuf-Struct) |  |  |






<a name="moth-server-v1-CreateUserResponse"></a>

### CreateUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-server-v1-User) |  |  |






<a name="moth-server-v1-DeleteUserRequest"></a>

### DeleteUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_id | [string](#string) |  |  |






<a name="moth-server-v1-DeleteUserResponse"></a>

### DeleteUserResponse







<a name="moth-server-v1-DisableUserRequest"></a>

### DisableUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_id | [string](#string) |  |  |






<a name="moth-server-v1-DisableUserResponse"></a>

### DisableUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-server-v1-User) |  |  |






<a name="moth-server-v1-EnableUserRequest"></a>

### EnableUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_id | [string](#string) |  |  |






<a name="moth-server-v1-EnableUserResponse"></a>

### EnableUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-server-v1-User) |  |  |






<a name="moth-server-v1-GetUserRequest"></a>

### GetUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_id | [string](#string) |  |  |






<a name="moth-server-v1-GetUserResponse"></a>

### GetUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-server-v1-User) |  |  |






<a name="moth-server-v1-ListUsersRequest"></a>

### ListUsersRequest







<a name="moth-server-v1-ListUsersResponse"></a>

### ListUsersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| users | [User](#moth-server-v1-User) | repeated |  |






<a name="moth-server-v1-RevokeUserSessionsRequest"></a>

### RevokeUserSessionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_id | [string](#string) |  |  |






<a name="moth-server-v1-RevokeUserSessionsResponse"></a>

### RevokeUserSessionsResponse







<a name="moth-server-v1-UpdateUserRequest"></a>

### UpdateUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_id | [string](#string) |  |  |
| display_name | [string](#string) | optional |  |
| avatar_url | [string](#string) | optional |  |
| custom_claims | [google.protobuf.Struct](#google-protobuf-Struct) |  | Replaces the whole custom claims object when set. |






<a name="moth-server-v1-UpdateUserResponse"></a>

### UpdateUserResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#moth-server-v1-User) |  |  |






<a name="moth-server-v1-User"></a>

### User
User is the full server-side view of an account.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| email | [string](#string) |  |  |
| email_verified | [bool](#bool) |  |  |
| display_name | [string](#string) |  |  |
| avatar_url | [string](#string) |  |  |
| custom_claims | [google.protobuf.Struct](#google-protobuf-Struct) |  | Embedded in the JWT under the `claims` claim. |
| disabled | [bool](#bool) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |





 

 

 


<a name="moth-server-v1-UserService"></a>

### UserService
UserService is programmatic user management for the developer&#39;s backend —
the moth counterpart of the Firebase Admin SDK. Authenticated with the
project secret key (`x-moth-key: sk_...`); always scoped to that project.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetUser | [GetUserRequest](#moth-server-v1-GetUserRequest) | [GetUserResponse](#moth-server-v1-GetUserResponse) |  |
| ListUsers | [ListUsersRequest](#moth-server-v1-ListUsersRequest) | [ListUsersResponse](#moth-server-v1-ListUsersResponse) |  |
| CreateUser | [CreateUserRequest](#moth-server-v1-CreateUserRequest) | [CreateUserResponse](#moth-server-v1-CreateUserResponse) | CreateUser provisions a user directly (e.g. invite-only projects). |
| UpdateUser | [UpdateUserRequest](#moth-server-v1-UpdateUserRequest) | [UpdateUserResponse](#moth-server-v1-UpdateUserResponse) | UpdateUser edits profile fields and custom_claims — the only way, besides the admin UI, to put roles/permissions into the JWT. Claim changes take effect on the next token refresh; pair with RevokeUserSessions to force it. |
| DisableUser | [DisableUserRequest](#moth-server-v1-DisableUserRequest) | [DisableUserResponse](#moth-server-v1-DisableUserResponse) | DisableUser blocks sign-in, refresh and introspection for the user. |
| EnableUser | [EnableUserRequest](#moth-server-v1-EnableUserRequest) | [EnableUserResponse](#moth-server-v1-EnableUserResponse) | EnableUser lifts a DisableUser block. |
| DeleteUser | [DeleteUserRequest](#moth-server-v1-DeleteUserRequest) | [DeleteUserResponse](#moth-server-v1-DeleteUserResponse) |  |
| RevokeUserSessions | [RevokeUserSessionsRequest](#moth-server-v1-RevokeUserSessionsRequest) | [RevokeUserSessionsResponse](#moth-server-v1-RevokeUserSessionsResponse) | RevokeUserSessions revokes every refresh token of the user. |

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

