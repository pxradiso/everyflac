; everyFLAC by pxradise - Windows installer
; Build with: makensis everyflac.nsi

Unicode true
ManifestDPIAware true

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "FileFunc.nsh"
!include "WinMessages.nsh"
!include "StrFunc.nsh"

${StrStr}
${UnStrRep}

; ---------------------------------------------------------------- metadata
!define APP_NAME       "everyFLAC"
!ifndef APP_VERSION
  !define APP_VERSION  "1.3.0"
!endif
!define APP_PUBLISHER  "pxradise"
!define APP_URL        "https://github.com/pxradiso/everyflac"
!define APP_EXE        "everyflac.exe"
!define UNINST_KEY     "Software\Microsoft\Windows\CurrentVersion\Uninstall\everyFLAC"
!define APP_KEY        "Software\everyFLAC"

Name "${APP_NAME}"
BrandingText "${APP_NAME} by ${APP_PUBLISHER}"
OutFile "everyflac-setup-windows-amd64.exe"
InstallDir "$LOCALAPPDATA\Programs\everyFLAC"
InstallDirRegKey HKCU "${APP_KEY}" "InstallDir"
RequestExecutionLevel user
SetCompressor /SOLID lzma
SetCompressorDictSize 32
Icon "everyflac.ico"
UninstallIcon "everyflac.ico"

VIProductVersion "${APP_VERSION}.0"
VIAddVersionKey "ProductName"     "${APP_NAME}"
VIAddVersionKey "ProductVersion"  "${APP_VERSION}"
VIAddVersionKey "CompanyName"     "${APP_PUBLISHER}"
VIAddVersionKey "FileDescription" "${APP_NAME} installer"
VIAddVersionKey "FileVersion"     "${APP_VERSION}.0"
VIAddVersionKey "LegalCopyright"  "MIT License"

; ---------------------------------------------------------------- UI
!define MUI_ICON "everyflac.ico"
!define MUI_UNICON "everyflac.ico"
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\${APP_EXE}"
!define MUI_FINISHPAGE_RUN_TEXT "$(RUN_APP)"
!define MUI_FINISHPAGE_LINK "${APP_URL}"
!define MUI_FINISHPAGE_LINK_LOCATION "${APP_URL}"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

; ---------------------------------------------------------------- strings
LangString SEC_APP_NAME      ${LANG_ENGLISH} "everyFLAC application and dependencies"
LangString SEC_DESKTOP_NAME  ${LANG_ENGLISH} "Desktop shortcut"

LangString DESC_APP          ${LANG_ENGLISH} "Installs everyFLAC, yt-dlp and ffmpeg. A Start Menu shortcut is created automatically."
LangString DESC_DESKTOP      ${LANG_ENGLISH} "Tick this to also place an everyFLAC shortcut on your desktop."

LangString RUN_APP           ${LANG_ENGLISH} "Launch everyFLAC"

LangString UNINST_CONFIG     ${LANG_ENGLISH} "Also remove your everyFLAC configuration and settings?"

; ---------------------------------------------------------------- install
Section "$(SEC_APP_NAME)" SEC_APP
  SectionIn RO

  SetOutPath "$INSTDIR"

  ; Native build when possible, x64 build (also runs on ARM64) otherwise.
  ; A 32-bit installer sees PROCESSOR_ARCHITECTURE=x86 on 64-bit Windows,
  ; so trust PROCESSOR_ARCHITEW6432 first and only fall back when it is empty.
  ReadEnvStr $0 "PROCESSOR_ARCHITEW6432"
  ${If} $0 == ""
    ReadEnvStr $0 "PROCESSOR_ARCHITECTURE"
  ${EndIf}
  ${If} $0 == "x86"
    MessageBox MB_OK|MB_ICONSTOP "everyFLAC requires 64-bit Windows 10 or 11."
    Abort
  ${EndIf}
  ${If} $0 == "ARM64"
    File /oname=${APP_EXE} ".payload/everyflac-arm64.exe"
  ${Else}
    File /oname=${APP_EXE} ".payload/everyflac-amd64.exe"
  ${EndIf}

  File ".payload/yt-dlp.exe"
  File ".payload/ffmpeg.exe"
  File "everyflac.ico"
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Start Menu shortcut (always created). It opens everyFLAC in a Command
  ; Prompt window so the TUI has a real terminal to run in.
  CreateDirectory "$SMPROGRAMS\everyFLAC"
  CreateShortcut "$SMPROGRAMS\everyFLAC\everyFLAC.lnk" "$SYSDIR\cmd.exe" '/k "$INSTDIR\${APP_EXE}"' "$INSTDIR\everyflac.ico" 0 SW_SHOWNORMAL "" "everyFLAC"
  CreateShortcut "$SMPROGRAMS\everyFLAC\Uninstall everyFLAC.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\uninstall.exe" 0

  ; Add the install directory to the per-user PATH once.
  ReadRegStr $0 HKCU "Environment" "Path"
  ${If} $0 == ""
    WriteRegExpandStr HKCU "Environment" "Path" "$INSTDIR"
  ${Else}
    ${StrStr} $1 "$0" "$INSTDIR"
    ${If} $1 == ""
      WriteRegExpandStr HKCU "Environment" "Path" "$INSTDIR;$0"
    ${EndIf}
  ${EndIf}
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

  ; Add/Remove Programs entry.
  WriteRegStr   HKCU "${APP_KEY}" "InstallDir" "$INSTDIR"
  WriteRegStr   HKCU "${UNINST_KEY}" "DisplayName"     "${APP_NAME}"
  WriteRegStr   HKCU "${UNINST_KEY}" "DisplayVersion"  "${APP_VERSION}"
  WriteRegStr   HKCU "${UNINST_KEY}" "Publisher"       "${APP_PUBLISHER}"
  WriteRegStr   HKCU "${UNINST_KEY}" "DisplayIcon"     "$INSTDIR\everyflac.ico"
  WriteRegStr   HKCU "${UNINST_KEY}" "URLInfoAbout"    "${APP_URL}"
  WriteRegStr   HKCU "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr   HKCU "${UNINST_KEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr   HKCU "${UNINST_KEY}" "QuietUninstallString" '"$INSTDIR\uninstall.exe" /S'
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoRepair" 1
  WriteRegDWORD HKCU "${UNINST_KEY}" "EstimatedSize" 120000

  ; Let Windows know the Start Menu changed.
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:ShellState" /TIMEOUT=5000
SectionEnd

Section /o "$(SEC_DESKTOP_NAME)" SEC_DESKTOP
  CreateShortcut "$DESKTOP\everyFLAC.lnk" "$SYSDIR\cmd.exe" '/k "$INSTDIR\${APP_EXE}"' "$INSTDIR\everyflac.ico" 0 SW_SHOWNORMAL "" "everyFLAC"
SectionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_APP} $(DESC_APP)
  !insertmacro MUI_DESCRIPTION_TEXT ${SEC_DESKTOP} $(DESC_DESKTOP)
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; ---------------------------------------------------------------- uninstall
Section "Uninstall"
  Delete "$INSTDIR\${APP_EXE}"
  Delete "$INSTDIR\yt-dlp.exe"
  Delete "$INSTDIR\ffmpeg.exe"
  Delete "$INSTDIR\everyflac.ico"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"

  Delete "$SMPROGRAMS\everyFLAC\everyFLAC.lnk"
  Delete "$SMPROGRAMS\everyFLAC\Uninstall everyFLAC.lnk"
  RMDir "$SMPROGRAMS\everyFLAC"
  Delete "$DESKTOP\everyFLAC.lnk"

  ; Remove our entry from the per-user PATH.
  ReadRegStr $0 HKCU "Environment" "Path"
  ${UnStrRep} $1 "$0" "$INSTDIR;" ""
  ${UnStrRep} $0 "$1" ";$INSTDIR" ""
  ${UnStrRep} $1 "$0" "$INSTDIR" ""
  WriteRegExpandStr HKCU "Environment" "Path" "$1"
  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:Environment" /TIMEOUT=5000

  DeleteRegKey HKCU "${UNINST_KEY}"
  DeleteRegKey HKCU "${APP_KEY}"

  MessageBox MB_YESNO|MB_ICONQUESTION "$(UNINST_CONFIG)" IDNO skip_config
    RMDir /r "$APPDATA\everyflac"
  skip_config:

  SendMessage ${HWND_BROADCAST} ${WM_WININICHANGE} 0 "STR:ShellState" /TIMEOUT=5000
SectionEnd
