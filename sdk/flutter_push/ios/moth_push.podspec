#
# moth's first-party APNs push-registration plugin. Produces the device
# credential the moth server stores; delivery and display stay app code.
#
Pod::Spec.new do |s|
  s.name             = 'moth_push'
  s.version          = '0.1.0'
  s.summary          = "moth's first-party APNs push-registration plugin."
  s.description      = <<-DESC
Native push registration for the moth Flutter SDK: UNUserNotificationCenter
authorization and the APNs device token, registered server-side by moth.
                       DESC
  s.homepage         = 'https://github.com/aloisdeniel/moth'
  s.license          = { :file => '../LICENSE' }
  s.author           = { 'Aloïs Deniel' => 'alois.deniel@gmail.com' }
  s.source           = { :path => '.' }
  s.source_files     = 'Classes/**/*'
  s.frameworks       = 'UserNotifications'
  s.dependency 'Flutter'
  s.platform         = :ios, '15.0'
  s.swift_version    = '5.9'
  s.pod_target_xcconfig = { 'DEFINES_MODULE' => 'YES' }
end
