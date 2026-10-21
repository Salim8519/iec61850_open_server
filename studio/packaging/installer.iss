; ScadaBench Windows installer (Inno Setup 6). Built by compile_protected.py:
;   ISCC.exe /DSourceDir=<dist\ScadaBench> /DOutputDir=<folder> /DAppVersion=0.1.0 installer.iss
; User data and the license in %LOCALAPPDATA%\ScadaBench are never removed.

#ifndef SourceDir
  #error SourceDir must point to the built ScadaBench folder
#endif
#ifndef OutputDir
  #define OutputDir "."
#endif
#ifndef AppVersion
  #define AppVersion "0.1.0"
#endif

[Setup]
AppId={{C3A61E52-9B47-4F0D-8E2A-5D17B6F4A9C8}
AppName=ScadaBench
AppVersion={#AppVersion}
AppPublisher=ScadaBench
DefaultDirName={autopf}\ScadaBench
DefaultGroupName=ScadaBench
DisableProgramGroupPage=yes
OutputDir={#OutputDir}
OutputBaseFilename=ScadaBench-Setup-{#AppVersion}
Compression=lzma2/max
SolidCompression=yes
ArchitecturesAllowed=x64
ArchitecturesInstallIn64BitMode=x64
; Per-user install by default (no administrator prompt); admins may choose all users.
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog commandline
WizardStyle=modern
DisableWelcomePage=no
SetupIconFile=..\branding\icon.ico
; Brand pictures (tools/build_installer_art.py), one size per display scaling level (change 0090).
WizardImageFile=..\branding\installer\wizard-164x314.bmp,..\branding\installer\wizard-192x386.bmp,..\branding\installer\wizard-246x459.bmp,..\branding\installer\wizard-273x556.bmp,..\branding\installer\wizard-328x628.bmp,..\branding\installer\wizard-355x669.bmp,..\branding\installer\wizard-410x797.bmp
WizardSmallImageFile=..\branding\installer\small-55x55.bmp,..\branding\installer\small-64x68.bmp,..\branding\installer\small-83x80.bmp,..\branding\installer\small-92x97.bmp,..\branding\installer\small-110x106.bmp,..\branding\installer\small-119x123.bmp,..\branding\installer\small-138x140.bmp
UninstallDisplayIcon={app}\ScadaBench.exe
CloseApplications=yes
SetupLogging=yes

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Shortcuts:"

[InstallDelete]
; Program files only: the Go and Python releases share this AppId and folder (ADR-0083), and
; an upgrade from one to the other must not keep the other's runtime files. User data and the
; license live in %LOCALAPPDATA% (or a data folder beside the exe), never in _internal.
Type: filesandordirs; Name: "{app}\_internal"

[Files]
Source: "{#SourceDir}\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\ScadaBench"; Filename: "{app}\ScadaBench.exe"; WorkingDir: "{app}"
Name: "{group}\Uninstall ScadaBench"; Filename: "{uninstallexe}"
Name: "{autodesktop}\ScadaBench"; Filename: "{app}\ScadaBench.exe"; WorkingDir: "{app}"; Tasks: desktopicon

[Run]
Filename: "{app}\ScadaBench.exe"; Description: "Start ScadaBench now"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
; Only files the installer created are removed; per-user data stays in %LOCALAPPDATA%.
Type: filesandordirs; Name: "{app}\_internal"

[Messages]
FinishedLabel=ScadaBench is installed. Your projects and license are kept in your user folder, so reinstalling or updating never deletes them.
