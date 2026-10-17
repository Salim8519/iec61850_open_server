; ScadaBench portable self-extractor (Inno Setup 6). Built by make_portable.py:
;   ISCC.exe /DSourceDir=<dist-protected\ScadaBench> /DOutputDir=<folder> /DAppVersion=0.1.6 portable.iss
; It only unpacks the sealed app folder where the user chooses and adds an empty data folder,
; which keeps projects beside ScadaBench.exe (portable mode). No registry, shortcuts or uninstaller.

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
AppId={{6F1D9C2B-4A7E-4B3F-9D5A-2C8E1B7F0A43}
AppName=ScadaBench portable
AppVersion={#AppVersion}
AppPublisher=ScadaBench
DefaultDirName={userdesktop}\ScadaBench
UsePreviousAppDir=no
DirExistsWarning=no
DisableProgramGroupPage=yes
Uninstallable=no
CreateUninstallRegKey=no
OutputDir={#OutputDir}
OutputBaseFilename=ScadaBench-Portable-{#AppVersion}
Compression=lzma2/max
SolidCompression=yes
ArchitecturesAllowed=x64
ArchitecturesInstallIn64BitMode=x64
PrivilegesRequired=lowest
WizardStyle=modern
SetupIconFile=..\branding\icon.ico
CloseApplications=yes

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Dirs]
; An existing data folder (and the projects in it) is kept; a new one switches on portable mode.
Name: "{app}\data"

[Files]
Source: "{#SourceDir}\*"; DestDir: "{app}"; Excludes: "data\*"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "PORTABLE.txt"; DestDir: "{app}"; Flags: ignoreversion

[Run]
Filename: "{app}\ScadaBench.exe"; Description: "Start ScadaBench now"; Flags: nowait postinstall skipifsilent

[Messages]
SetupWindowTitle=ScadaBench portable %1
WelcomeLabel2=This unpacks ScadaBench into one folder you choose. Nothing is installed in Windows, and your projects stay in that folder.
SelectDirLabel3=Choose where to put the ScadaBench folder, for example the Desktop or a USB stick. To update a portable copy, choose its folder again: your projects are kept.
FinishedLabel=ScadaBench is ready in the folder you chose. Start ScadaBench.exe from there; your projects stay in its data folder.
