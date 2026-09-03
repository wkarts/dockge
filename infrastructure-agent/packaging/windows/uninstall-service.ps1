[CmdletBinding()]
param(
    [switch]$PurgeData,
    [string]$InstallDir = "$env:ProgramFiles\InfrastructureAgent",
    [string]$ConfigDir = "$env:ProgramData\InfrastructureAgent"
)

$ErrorActionPreference = 'Stop'
$serviceName = 'InfrastructureAgent'

$id = [Security.Principal.WindowsIdentity]::GetCurrent()
$principal = New-Object Security.Principal.WindowsPrincipal($id)
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw 'Execute a desinstalação como Administrador.'
}

Write-Host 'Generic Infrastructure Agent — Desinstalação' -ForegroundColor Cyan
Write-Host ('─' * 56) -ForegroundColor DarkGray
$service = Get-Service -Name $serviceName -ErrorAction SilentlyContinue
if ($service) {
    Stop-Service $serviceName -Force -ErrorAction SilentlyContinue
    sc.exe delete $serviceName | Out-Null
    Write-Host '✓ Serviço removido.' -ForegroundColor Green
}
Remove-Item $InstallDir -Recurse -Force -ErrorAction SilentlyContinue
Write-Host '✓ Binários removidos.' -ForegroundColor Green

if ($PurgeData) {
    Remove-Item $ConfigDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Host '✓ Configurações e credenciais removidas (--PurgeData).' -ForegroundColor Green
} else {
    Write-Host "● Dados preservados em $ConfigDir" -ForegroundColor Yellow
}
