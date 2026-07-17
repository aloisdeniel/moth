// This is a generated file - do not edit.
//
// Generated from moth/storage/v1/storage.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports
// ignore_for_file: unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use legalLinksDescriptor instead')
const LegalLinks$json = {
  '1': 'LegalLinks',
  '2': [
    {'1': 'terms_url', '3': 1, '4': 1, '5': 9, '10': 'termsUrl'},
    {'1': 'privacy_url', '3': 2, '4': 1, '5': 9, '10': 'privacyUrl'},
  ],
};

/// Descriptor for `LegalLinks`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List legalLinksDescriptor = $convert.base64Decode(
    'CgpMZWdhbExpbmtzEhsKCXRlcm1zX3VybBgBIAEoCVIIdGVybXNVcmwSHwoLcHJpdmFjeV91cm'
    'wYAiABKAlSCnByaXZhY3lVcmw=');

@$core.Deprecated('Use themeColorsDescriptor instead')
const ThemeColors$json = {
  '1': 'ThemeColors',
  '2': [
    {'1': 'primary', '3': 1, '4': 1, '5': 9, '10': 'primary'},
    {'1': 'on_primary', '3': 2, '4': 1, '5': 9, '10': 'onPrimary'},
    {'1': 'background', '3': 3, '4': 1, '5': 9, '10': 'background'},
    {'1': 'on_background', '3': 4, '4': 1, '5': 9, '10': 'onBackground'},
    {'1': 'surface', '3': 5, '4': 1, '5': 9, '10': 'surface'},
    {'1': 'on_surface', '3': 6, '4': 1, '5': 9, '10': 'onSurface'},
    {'1': 'error', '3': 7, '4': 1, '5': 9, '10': 'error'},
    {'1': 'on_error', '3': 8, '4': 1, '5': 9, '10': 'onError'},
  ],
};

/// Descriptor for `ThemeColors`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List themeColorsDescriptor = $convert.base64Decode(
    'CgtUaGVtZUNvbG9ycxIYCgdwcmltYXJ5GAEgASgJUgdwcmltYXJ5Eh0KCm9uX3ByaW1hcnkYAi'
    'ABKAlSCW9uUHJpbWFyeRIeCgpiYWNrZ3JvdW5kGAMgASgJUgpiYWNrZ3JvdW5kEiMKDW9uX2Jh'
    'Y2tncm91bmQYBCABKAlSDG9uQmFja2dyb3VuZBIYCgdzdXJmYWNlGAUgASgJUgdzdXJmYWNlEh'
    '0KCm9uX3N1cmZhY2UYBiABKAlSCW9uU3VyZmFjZRIUCgVlcnJvchgHIAEoCVIFZXJyb3ISGQoI'
    'b25fZXJyb3IYCCABKAlSB29uRXJyb3I=');

@$core.Deprecated('Use themeColorOverridesDescriptor instead')
const ThemeColorOverrides$json = {
  '1': 'ThemeColorOverrides',
  '2': [
    {'1': 'primary', '3': 1, '4': 1, '5': 9, '10': 'primary'},
    {'1': 'on_primary', '3': 2, '4': 1, '5': 9, '10': 'onPrimary'},
    {'1': 'background', '3': 3, '4': 1, '5': 9, '10': 'background'},
    {'1': 'on_background', '3': 4, '4': 1, '5': 9, '10': 'onBackground'},
    {'1': 'surface', '3': 5, '4': 1, '5': 9, '10': 'surface'},
    {'1': 'on_surface', '3': 6, '4': 1, '5': 9, '10': 'onSurface'},
    {'1': 'error', '3': 7, '4': 1, '5': 9, '10': 'error'},
    {'1': 'on_error', '3': 8, '4': 1, '5': 9, '10': 'onError'},
  ],
};

/// Descriptor for `ThemeColorOverrides`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List themeColorOverridesDescriptor = $convert.base64Decode(
    'ChNUaGVtZUNvbG9yT3ZlcnJpZGVzEhgKB3ByaW1hcnkYASABKAlSB3ByaW1hcnkSHQoKb25fcH'
    'JpbWFyeRgCIAEoCVIJb25QcmltYXJ5Eh4KCmJhY2tncm91bmQYAyABKAlSCmJhY2tncm91bmQS'
    'IwoNb25fYmFja2dyb3VuZBgEIAEoCVIMb25CYWNrZ3JvdW5kEhgKB3N1cmZhY2UYBSABKAlSB3'
    'N1cmZhY2USHQoKb25fc3VyZmFjZRgGIAEoCVIJb25TdXJmYWNlEhQKBWVycm9yGAcgASgJUgVl'
    'cnJvchIZCghvbl9lcnJvchgIIAEoCVIHb25FcnJvcg==');

@$core.Deprecated('Use themeTypographyDescriptor instead')
const ThemeTypography$json = {
  '1': 'ThemeTypography',
  '2': [
    {'1': 'font_family', '3': 1, '4': 1, '5': 9, '10': 'fontFamily'},
    {'1': 'scale', '3': 2, '4': 1, '5': 1, '10': 'scale'},
  ],
};

/// Descriptor for `ThemeTypography`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List themeTypographyDescriptor = $convert.base64Decode(
    'Cg9UaGVtZVR5cG9ncmFwaHkSHwoLZm9udF9mYW1pbHkYASABKAlSCmZvbnRGYW1pbHkSFAoFc2'
    'NhbGUYAiABKAFSBXNjYWxl');

@$core.Deprecated('Use themeSpacingDescriptor instead')
const ThemeSpacing$json = {
  '1': 'ThemeSpacing',
  '2': [
    {'1': 'unit', '3': 1, '4': 1, '5': 5, '10': 'unit'},
  ],
};

/// Descriptor for `ThemeSpacing`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List themeSpacingDescriptor =
    $convert.base64Decode('CgxUaGVtZVNwYWNpbmcSEgoEdW5pdBgBIAEoBVIEdW5pdA==');

@$core.Deprecated('Use themeShapeDescriptor instead')
const ThemeShape$json = {
  '1': 'ThemeShape',
  '2': [
    {'1': 'corner_radius', '3': 1, '4': 1, '5': 5, '10': 'cornerRadius'},
  ],
};

/// Descriptor for `ThemeShape`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List themeShapeDescriptor = $convert.base64Decode(
    'CgpUaGVtZVNoYXBlEiMKDWNvcm5lcl9yYWRpdXMYASABKAVSDGNvcm5lclJhZGl1cw==');

@$core.Deprecated('Use themeLogoDescriptor instead')
const ThemeLogo$json = {
  '1': 'ThemeLogo',
  '2': [
    {'1': 'light', '3': 1, '4': 1, '5': 9, '10': 'light'},
    {'1': 'dark', '3': 2, '4': 1, '5': 9, '10': 'dark'},
  ],
};

/// Descriptor for `ThemeLogo`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List themeLogoDescriptor = $convert.base64Decode(
    'CglUaGVtZUxvZ28SFAoFbGlnaHQYASABKAlSBWxpZ2h0EhIKBGRhcmsYAiABKAlSBGRhcms=');

@$core.Deprecated('Use storedThemeDescriptor instead')
const StoredTheme$json = {
  '1': 'StoredTheme',
  '2': [
    {'1': 'version', '3': 1, '4': 1, '5': 5, '10': 'version'},
    {
      '1': 'colors',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.moth.storage.v1.ThemeColors',
      '10': 'colors'
    },
    {
      '1': 'dark_colors',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.moth.storage.v1.ThemeColorOverrides',
      '10': 'darkColors'
    },
    {
      '1': 'typography',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.moth.storage.v1.ThemeTypography',
      '10': 'typography'
    },
    {
      '1': 'spacing',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.moth.storage.v1.ThemeSpacing',
      '10': 'spacing'
    },
    {
      '1': 'shape',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.moth.storage.v1.ThemeShape',
      '10': 'shape'
    },
    {
      '1': 'logo',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.moth.storage.v1.ThemeLogo',
      '10': 'logo'
    },
    {
      '1': 'legal',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.moth.storage.v1.LegalLinks',
      '10': 'legal'
    },
  ],
};

/// Descriptor for `StoredTheme`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storedThemeDescriptor = $convert.base64Decode(
    'CgtTdG9yZWRUaGVtZRIYCgd2ZXJzaW9uGAEgASgFUgd2ZXJzaW9uEjQKBmNvbG9ycxgCIAEoCz'
    'IcLm1vdGguc3RvcmFnZS52MS5UaGVtZUNvbG9yc1IGY29sb3JzEkUKC2RhcmtfY29sb3JzGAMg'
    'ASgLMiQubW90aC5zdG9yYWdlLnYxLlRoZW1lQ29sb3JPdmVycmlkZXNSCmRhcmtDb2xvcnMSQA'
    'oKdHlwb2dyYXBoeRgEIAEoCzIgLm1vdGguc3RvcmFnZS52MS5UaGVtZVR5cG9ncmFwaHlSCnR5'
    'cG9ncmFwaHkSNwoHc3BhY2luZxgFIAEoCzIdLm1vdGguc3RvcmFnZS52MS5UaGVtZVNwYWNpbm'
    'dSB3NwYWNpbmcSMQoFc2hhcGUYBiABKAsyGy5tb3RoLnN0b3JhZ2UudjEuVGhlbWVTaGFwZVIF'
    'c2hhcGUSLgoEbG9nbxgHIAEoCzIaLm1vdGguc3RvcmFnZS52MS5UaGVtZUxvZ29SBGxvZ28SMQ'
    'oFbGVnYWwYCCABKAsyGy5tb3RoLnN0b3JhZ2UudjEuTGVnYWxMaW5rc1IFbGVnYWw=');

@$core.Deprecated('Use storedPaywallDescriptor instead')
const StoredPaywall$json = {
  '1': 'StoredPaywall',
  '2': [
    {'1': 'version', '3': 1, '4': 1, '5': 5, '10': 'version'},
    {'1': 'headline', '3': 2, '4': 1, '5': 9, '10': 'headline'},
    {'1': 'subtitle', '3': 3, '4': 1, '5': 9, '10': 'subtitle'},
    {'1': 'benefits', '3': 4, '4': 3, '5': 9, '10': 'benefits'},
    {'1': 'offering', '3': 5, '4': 1, '5': 9, '10': 'offering'},
    {
      '1': 'highlighted_identifier',
      '3': 6,
      '4': 1,
      '5': 9,
      '10': 'highlightedIdentifier'
    },
    {'1': 'layout', '3': 7, '4': 1, '5': 9, '10': 'layout'},
    {
      '1': 'legal',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.moth.storage.v1.LegalLinks',
      '10': 'legal'
    },
  ],
};

/// Descriptor for `StoredPaywall`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storedPaywallDescriptor = $convert.base64Decode(
    'Cg1TdG9yZWRQYXl3YWxsEhgKB3ZlcnNpb24YASABKAVSB3ZlcnNpb24SGgoIaGVhZGxpbmUYAi'
    'ABKAlSCGhlYWRsaW5lEhoKCHN1YnRpdGxlGAMgASgJUghzdWJ0aXRsZRIaCghiZW5lZml0cxgE'
    'IAMoCVIIYmVuZWZpdHMSGgoIb2ZmZXJpbmcYBSABKAlSCG9mZmVyaW5nEjUKFmhpZ2hsaWdodG'
    'VkX2lkZW50aWZpZXIYBiABKAlSFWhpZ2hsaWdodGVkSWRlbnRpZmllchIWCgZsYXlvdXQYByAB'
    'KAlSBmxheW91dBIxCgVsZWdhbBgIIAEoCzIbLm1vdGguc3RvcmFnZS52MS5MZWdhbExpbmtzUg'
    'VsZWdhbA==');

@$core.Deprecated('Use copyLocaleMessagesDescriptor instead')
const CopyLocaleMessages$json = {
  '1': 'CopyLocaleMessages',
  '2': [
    {
      '1': 'messages',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.moth.storage.v1.CopyLocaleMessages.MessagesEntry',
      '10': 'messages'
    },
  ],
  '3': [CopyLocaleMessages_MessagesEntry$json],
};

@$core.Deprecated('Use copyLocaleMessagesDescriptor instead')
const CopyLocaleMessages_MessagesEntry$json = {
  '1': 'MessagesEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `CopyLocaleMessages`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List copyLocaleMessagesDescriptor = $convert.base64Decode(
    'ChJDb3B5TG9jYWxlTWVzc2FnZXMSTQoIbWVzc2FnZXMYASADKAsyMS5tb3RoLnN0b3JhZ2Uudj'
    'EuQ29weUxvY2FsZU1lc3NhZ2VzLk1lc3NhZ2VzRW50cnlSCG1lc3NhZ2VzGjsKDU1lc3NhZ2Vz'
    'RW50cnkSEAoDa2V5GAEgASgJUgNrZXkSFAoFdmFsdWUYAiABKAlSBXZhbHVlOgI4AQ==');

@$core.Deprecated('Use storedCopyDescriptor instead')
const StoredCopy$json = {
  '1': 'StoredCopy',
  '2': [
    {
      '1': 'locales',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.moth.storage.v1.StoredCopy.LocalesEntry',
      '10': 'locales'
    },
  ],
  '3': [StoredCopy_LocalesEntry$json],
};

@$core.Deprecated('Use storedCopyDescriptor instead')
const StoredCopy_LocalesEntry$json = {
  '1': 'LocalesEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {
      '1': 'value',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.moth.storage.v1.CopyLocaleMessages',
      '10': 'value'
    },
  ],
  '7': {'7': true},
};

/// Descriptor for `StoredCopy`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List storedCopyDescriptor = $convert.base64Decode(
    'CgpTdG9yZWRDb3B5EkIKB2xvY2FsZXMYASADKAsyKC5tb3RoLnN0b3JhZ2UudjEuU3RvcmVkQ2'
    '9weS5Mb2NhbGVzRW50cnlSB2xvY2FsZXMaXwoMTG9jYWxlc0VudHJ5EhAKA2tleRgBIAEoCVID'
    'a2V5EjkKBXZhbHVlGAIgASgLMiMubW90aC5zdG9yYWdlLnYxLkNvcHlMb2NhbGVNZXNzYWdlc1'
    'IFdmFsdWU6AjgB');

@$core.Deprecated('Use cacheEnvelopeDescriptor instead')
const CacheEnvelope$json = {
  '1': 'CacheEnvelope',
  '2': [
    {'1': 'payload', '3': 1, '4': 1, '5': 12, '10': 'payload'},
    {'1': 'revision', '3': 2, '4': 1, '5': 9, '10': 'revision'},
    {'1': 'locale', '3': 3, '4': 1, '5': 9, '10': 'locale'},
    {
      '1': 'fetched_at_unix_ms',
      '3': 4,
      '4': 1,
      '5': 3,
      '10': 'fetchedAtUnixMs'
    },
  ],
};

/// Descriptor for `CacheEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cacheEnvelopeDescriptor = $convert.base64Decode(
    'Cg1DYWNoZUVudmVsb3BlEhgKB3BheWxvYWQYASABKAxSB3BheWxvYWQSGgoIcmV2aXNpb24YAi'
    'ABKAlSCHJldmlzaW9uEhYKBmxvY2FsZRgDIAEoCVIGbG9jYWxlEisKEmZldGNoZWRfYXRfdW5p'
    'eF9tcxgEIAEoA1IPZmV0Y2hlZEF0VW5peE1z');
