#define MyAppName "Generic Infrastructure Agent"
#ifndef MyAppVersion
  #define MyAppVersion "0.2.0"
#endif
#ifndef MySourceDir
  #define MySourceDir "."
#endif
#ifndef MyOutputDir
  #define MyOutputDir "."
#endif
#ifndef MyPackagingDir
  #define MyPackagingDir "."
#endif

[Setup]
AppId={{50A56494-7281-4AF9-B881-E0F2B2747D33}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher=wkarts
AppPublisherURL=https://github.com/wkarts/dockge
DefaultDirName={autopf}\InfrastructureAgent
DisableProgramGroupPage=yes
PrivilegesRequired=admin
ArchitecturesAllowed=arm64
ArchitecturesInstallIn64BitMode=arm64
OutputDir={#MyOutputDir}
OutputBaseFilename=infrastructure-agent-setup-{#MyAppVersion}-windows-arm64
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
UninstallDisplayName={#MyAppName}

[Files]
Source: "{#MySourceDir}\infra-agent-windows-arm64.exe"; DestDir: "{app}"; DestName: "infra-agent.exe"; Flags: ignoreversion
Source: "{#MyPackagingDir}\install-service.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#MyPackagingDir}\configure-ui.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#MyPackagingDir}\uninstall-service.ps1"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#MyPackagingDir}\install.cmd"; DestDir: "{app}"; Flags: ignoreversion

[Dirs]
Name: "{commonappdata}\InfrastructureAgent"; Permissions: users-readexec
Name: "{commonappdata}\InfrastructureAgent\secrets"; Permissions: admins-full system-full

[Run]
Filename: "powershell.exe"; Parameters: "-NoLogo -NoProfile -ExecutionPolicy Bypass -File ""{app}\install-service.ps1"" -Action Install -InstallDir ""{app}"" -ConfigDir ""{commonappdata}\InfrastructureAgent"""; Flags: runhidden waituntilterminated; StatusMsg: "Registrando Infrastructure Agent como serviço..."
Filename: "powershell.exe"; Parameters: "-NoLogo -NoProfile -ExecutionPolicy Bypass -File ""{app}\configure-ui.ps1"" -InstallDir ""{app}"" -ConfigDir ""{commonappdata}\InfrastructureAgent"""; Description: "Configurar e vincular esta máquina agora"; Flags: postinstall skipifsilent nowait

[UninstallRun]
Filename: "powershell.exe"; Parameters: "-NoLogo -NoProfile -ExecutionPolicy Bypass -File ""{app}\uninstall-service.ps1"" -InstallDir ""{app}"" -ConfigDir ""{commonappdata}\InfrastructureAgent"""; Flags: runhidden waituntilterminated

[Icons]
Name: "{autoprograms}\Infrastructure Agent\Configurar Agent"; Filename: "powershell.exe"; Parameters: "-NoLogo -NoProfile -ExecutionPolicy Bypass -File ""{app}\configure-ui.ps1"" -InstallDir ""{app}"" -ConfigDir ""{commonappdata}\InfrastructureAgent"""
Name: "{autoprograms}\Infrastructure Agent\Administrador CLI"; Filename: "powershell.exe"; Parameters: "-NoLogo -NoProfile -ExecutionPolicy Bypass -File ""{app}\install-service.ps1"" -InstallDir ""{app}"" -ConfigDir ""{commonappdata}\InfrastructureAgent"""
