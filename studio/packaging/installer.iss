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
SetupIconFile=..\branding\icon.ico
UninstallDisplayIcon={app}\ScadaBench.exe
CloseApplications=yes
SetupLogging=yes

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Shortcuts:"

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
