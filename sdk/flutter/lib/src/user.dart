/// The signed-in user's own account, as exposed to the app.
class MothUser {
  const MothUser({
    required this.id,
    required this.email,
    required this.emailVerified,
    this.displayName,
    this.avatarUrl,
    this.createTime,
    this.claims = const {},
  });

  factory MothUser.fromJson(Map<String, Object?> json) => MothUser(
    id: json['id'] as String,
    email: json['email'] as String,
    emailVerified: json['email_verified'] as bool? ?? false,
    displayName: json['display_name'] as String?,
    avatarUrl: json['avatar_url'] as String?,
    createTime: switch (json['create_time']) {
      final String s => DateTime.tryParse(s),
      _ => null,
    },
    claims: switch (json['claims']) {
      final Map<String, Object?> m => m,
      _ => const {},
    },
  );

  final String id;
  final String email;
  final bool emailVerified;
  final String? displayName;
  final String? avatarUrl;
  final DateTime? createTime;

  /// The project-assigned custom claims (roles, permissions, ...) decoded
  /// from the access token's `claims` claim — without signature
  /// verification. Use them for client-side gating only; the developer's
  /// backend must verify the JWT against the project JWKS and remains the
  /// authority.
  final Map<String, Object?> claims;

  MothUser copyWith({Map<String, Object?>? claims}) => MothUser(
    id: id,
    email: email,
    emailVerified: emailVerified,
    displayName: displayName,
    avatarUrl: avatarUrl,
    createTime: createTime,
    claims: claims ?? this.claims,
  );

  Map<String, Object?> toJson() => {
    'id': id,
    'email': email,
    'email_verified': emailVerified,
    if (displayName != null) 'display_name': displayName,
    if (avatarUrl != null) 'avatar_url': avatarUrl,
    if (createTime != null)
      'create_time': createTime!.toUtc().toIso8601String(),
    if (claims.isNotEmpty) 'claims': claims,
  };

  @override
  String toString() => 'MothUser($id, $email)';
}
