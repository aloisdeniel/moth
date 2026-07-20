#
# moth's first-party StoreKit 2 billing plugin. Requires iOS 15 (StoreKit 2
# only — no StoreKit 1 fallback; the server validates the transaction JWS).
#
Pod::Spec.new do |s|
  s.name             = 'moth_billing'
  s.version          = '0.1.0'
  s.summary          = "moth's first-party StoreKit 2 billing plugin."
  s.description      = <<-DESC
Native billing for the moth Flutter SDK: StoreKit 2 subscriptions whose
verified transaction JWS is validated server-side by moth.
                       DESC
  s.homepage         = 'https://github.com/aloisdeniel/moth'
  s.license          = { :file => '../LICENSE' }
  s.author           = { 'Aloïs Deniel' => 'alois.deniel@gmail.com' }
  s.source           = { :path => '.' }
  s.source_files     = 'Classes/**/*'
  s.dependency 'Flutter'
  s.platform         = :ios, '15.0'
  s.swift_version    = '5.9'
  s.pod_target_xcconfig = { 'DEFINES_MODULE' => 'YES' }
end
