param([string]$GoExecutable = 'go')
$ErrorActionPreference = 'Stop'
$env:CGO_ENABLED = '0'
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:GOTOOLCHAIN = 'local'
Push-Location $PSScriptRoot
try {
    & $GoExecutable test -mod=vendor ./...
    if ($LASTEXITCODE -ne 0) { throw 'Las pruebas han fallado.' }
    New-Item -ItemType Directory -Force -Path 'dist' | Out-Null
    & $GoExecutable build -mod=vendor -trimpath -ldflags '-s -w -buildid=' -o 'dist\NexoChess.exe' .
    if ($LASTEXITCODE -ne 0) { throw 'La compilación ha fallado.' }
    Write-Host 'Creado: dist\NexoChess.exe'
} finally { Pop-Location }
