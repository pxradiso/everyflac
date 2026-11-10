# everyFLAC by pxradise - PowerShell entry point.
#
# The real installer lives in setup.bat so that double-clicking and
# PowerShell behave exactly the same. This wrapper exists so that
# `.\setup.ps1` also works, as documented in the README.

$ErrorActionPreference = 'Stop'

$bat = Join-Path $PSScriptRoot 'setup.bat'
if (-not (Test-Path -LiteralPath $bat)) {
    Write-Host "setup.bat was not found next to setup.ps1." -ForegroundColor Red
    exit 1
}

& cmd.exe /c "`"$bat`""
exit $LASTEXITCODE
