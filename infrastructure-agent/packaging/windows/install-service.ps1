[CmdletBinding()]
param(
    [ValidateSet('Menu','Install','Configure','Start','Doctor','Uninstall')]
    [string]$Action = 'Menu',
    [string]$InstallDir = "$env:ProgramFiles\InfrastructureAgent",
    [string]$ConfigDir = "$env:ProgramData\InfrastructureAgent"
)

$ErrorActionPreference = 'Stop'
$ServiceName = 'InfrastructureAgent'
$ConfigFile = Join-Path $ConfigDir 'agent.json'

function Write-Title {
    Clear-Host
    Write-Host '╔══════════════════════════════════════════════════════════════════╗' -ForegroundColor Cyan
    Write-Host '║              GENERIC INFRASTRUCTURE AGENT                       ║' -ForegroundColor Cyan
    Write-Host '║        Bootstrap • Enrollment • Docker/Dockge Bridge            ║' -ForegroundColor Cyan
    Write-Host '╚══════════════════════════════════════════════════════════════════╝' -ForegroundColor Cyan
    Write-Host 'Agente genérico para múltiplos Control Planes e plataformas.' -ForegroundColor DarkGray
    Write-Host
}
function Write-Ok([string]$Text) { Write-Host "✓ $Text" -ForegroundColor Green }
function Write-Info([string]$Text) { Write-Host "● $Text" -ForegroundColor Cyan }
function Write-Warn([string]$Text) { Write-Host "! $Text" -ForegroundColor Yellow }
function Test-Admin {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    $p = New-Object Security.Principal.WindowsPrincipal($id)
    return $p.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}
function Require-Admin {
    if (-not (Test-Admin)) { throw 'Execute este instalador como Administrador.' }
}
function Get-PlainText([Security.SecureString]$Secure) {
    $ptr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($Secure)
    try { [Runtime.InteropServices.Marshal]::PtrToStringBSTR($ptr) }
    finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($ptr) }
}
function Find-AgentBinary {
    $candidates = @(
        (Join-Path $PSScriptRoot 'infra-agent.exe'),
        (Join-Path $PSScriptRoot 'infra-agent-windows-amd64.exe'),
        (Join-Path $PSScriptRoot 'infra-agent-windows-arm64.exe'),
        (Join-Path $InstallDir 'infra-agent.exe')
    )
    foreach ($candidate in $candidates) { if (Test-Path $candidate) { return $candidate } }
    throw 'infra-agent.exe não foi encontrado neste pacote.'
}
function Protect-ConfigDirectory {
    New-Item -ItemType Directory -Force -Path $ConfigDir,(Join-Path $ConfigDir 'secrets'),(Join-Path $ConfigDir 'data') | Out-Null
    # SIDs tornam a regra independente do idioma do Windows:
    # S-1-5-18 = LocalSystem; S-1-5-32-544 = Administrators.
    & icacls.exe $ConfigDir '/inheritance:r' '/grant:r' '*S-1-5-18:(OI)(CI)F' '*S-1-5-32-544:(OI)(CI)F' '/T' '/C' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Falha ao restringir ACL da configuração.' }
}
function Set-ServicePolicy {
    & sc.exe config $ServiceName start= delayed-auto | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Falha ao configurar delayed auto-start.' }
    & sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/15000/restart/60000 | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Falha ao configurar recuperação automática.' }
    & sc.exe failureflag $ServiceName 1 | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Falha ao habilitar ações de recuperação para falhas não-crash.' }
}
function Show-Environment {
    Write-Title
    Write-Host 'Diagnóstico rápido do host' -ForegroundColor White
    Write-Host ('─' * 68) -ForegroundColor DarkGray
    $docker = try { (& docker --version 2>$null) } catch { 'não detectado' }
    $compose = try { (& docker compose version 2>$null) } catch { 'não detectado' }
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    $agent = if (Test-Path (Join-Path $InstallDir 'infra-agent.exe')) { & (Join-Path $InstallDir 'infra-agent.exe') version } else { 'não instalado' }
    '{0,-24}{1}' -f '  Windows', [Environment]::OSVersion.VersionString | Write-Host
    '{0,-24}{1}' -f '  Arquitetura', $env:PROCESSOR_ARCHITECTURE | Write-Host
    '{0,-24}{1}' -f '  Docker', $docker | Write-Host
    '{0,-24}{1}' -f '  Compose', $compose | Write-Host
    '{0,-24}{1}' -f '  Agent', $agent | Write-Host
    '{0,-24}{1}' -f '  Serviço', $(if ($service) { $service.Status } else { 'não registrado' }) | Write-Host
    '{0,-24}{1}' -f '  Configuração', $(if (Test-Path $ConfigFile) { $ConfigFile } else { 'pendente' }) | Write-Host
    Write-Host ('─' * 68) -ForegroundColor DarkGray
}
function Install-Agent {
    Require-Admin
    $source = Find-AgentBinary
    Write-Info 'Preparando diretórios persistentes e ACLs...'
    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Protect-ConfigDirectory
    Copy-Item $source (Join-Path $InstallDir 'infra-agent.exe') -Force
    $existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if ($existing) {
        Stop-Service $ServiceName -Force -ErrorAction SilentlyContinue
        & sc.exe delete $ServiceName | Out-Null
        Start-Sleep -Milliseconds 1200
    }
    $binPath = '\"{0}\" --config \"{1}\" run' -f (Join-Path $InstallDir 'infra-agent.exe'), $ConfigFile
    & sc.exe create $ServiceName binPath= $binPath start= demand DisplayName= 'Infrastructure Agent' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Falha ao registrar Windows Service.' }
    & sc.exe description $ServiceName 'Generic REST infrastructure enrollment and typed deployment reconciliation agent' | Out-Null
    Set-ServicePolicy
    Write-Ok "Agent instalado em $InstallDir"
    Write-Ok 'Windows Service registrado com delayed auto-start e recuperação automática.'
    Write-Ok "Configuração protegida em $ConfigDir"
}
function Configure-Agent {
    Require-Admin
    $agent = Join-Path $InstallDir 'infra-agent.exe'
    if (-not (Test-Path $agent)) { throw 'Instale o Agent antes de configurar.' }
    Protect-ConfigDirectory
    Write-Title
    Write-Host 'Configuração e vínculo com Control Plane' -ForegroundColor White
    Write-Host 'Vínculos existentes são preservados; informar o mesmo nome atualiza somente aquele Control Plane.' -ForegroundColor DarkGray
    $controller = Read-Host 'Nome lógico do Control Plane [control-plane]'; if ([string]::IsNullOrWhiteSpace($controller)) { $controller='control-plane' }
    $url = Read-Host 'URL REST/HTTPS do Control Plane [https://api.example.com]'; if ([string]::IsNullOrWhiteSpace($url)) { $url='https://api.example.com' }
    $prefixes = Read-Host 'Prefixos de deployments permitidos, separados por vírgula'
    $dockge = Read-Host 'Dockge API local [http://127.0.0.1:5001]'; if ([string]::IsNullOrWhiteSpace($dockge)) { $dockge='http://127.0.0.1:5001' }
    $environment = Read-Host 'Ambiente/label [production]'; if ([string]::IsNullOrWhiteSpace($environment)) { $environment='production' }
    $enrollment = Get-PlainText (Read-Host 'Credencial de enrollment (entrada protegida)' -AsSecureString)
    $dockgeCredential = Get-PlainText (Read-Host 'Credencial da Dockge API (Enter para preservar a existente)' -AsSecureString)
    try {
        $env:INFRA_AGENT_CONTROLLER_NAME=$controller
        $env:INFRA_AGENT_CONTROLLER_URL=$url
        $env:INFRA_AGENT_ALLOWED_PREFIXES=$prefixes
        $env:INFRA_AGENT_DOCKGE_URL=$dockge
        $env:INFRA_AGENT_ENROLLMENT_TOKEN=$enrollment
        $env:INFRA_AGENT_DOCKGE_TOKEN=$dockgeCredential
        $env:INFRA_AGENT_ENVIRONMENT=$environment
        $env:INFRA_AGENT_DATA_DIR=(Join-Path $ConfigDir 'data')
        & $agent --config $ConfigFile configure
        if ($LASTEXITCODE -ne 0) { throw 'Falha ao materializar configuração.' }
        Protect-ConfigDirectory
        Write-Ok 'Configuração criada; credenciais permanecem separadas do agent.json.'
        if (-not [string]::IsNullOrWhiteSpace($enrollment)) {
            Write-Info 'Realizando enrollment...'
            & $agent --config $ConfigFile enroll
            if ($LASTEXITCODE -eq 0) { Write-Ok 'Enrollment concluído e credencial bootstrap removida.' } else { Write-Warn 'Enrollment pendente; configuração foi preservada.' }
        }
        if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
            Restart-Service $ServiceName -Force -ErrorAction SilentlyContinue
        }
    } finally {
        'INFRA_AGENT_CONTROLLER_NAME','INFRA_AGENT_CONTROLLER_URL','INFRA_AGENT_ALLOWED_PREFIXES','INFRA_AGENT_DOCKGE_URL','INFRA_AGENT_ENROLLMENT_TOKEN','INFRA_AGENT_DOCKGE_TOKEN','INFRA_AGENT_ENVIRONMENT','INFRA_AGENT_DATA_DIR' | ForEach-Object { Remove-Item "Env:$_" -ErrorAction SilentlyContinue }
    }
}
function Start-Agent {
    Require-Admin
    $service = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if (-not $service) { throw 'Serviço não registrado. Execute Instalar/Reparar primeiro.' }
    Set-ServicePolicy
    if ($service.Status -eq 'Running') { Restart-Service $ServiceName -Force } else { Start-Service $ServiceName }
    Start-Sleep -Milliseconds 900
    Get-Service $ServiceName | Format-Table -AutoSize
}
function Doctor-Agent {
    $agent = Join-Path $InstallDir 'infra-agent.exe'
    if (-not (Test-Path $agent)) { throw 'Agent não instalado.' }
    & $agent --config $ConfigFile doctor
}
function Uninstall-Agent {
    Require-Admin
    Write-Warn 'A desinstalação preservará configuração, credenciais e dados persistentes.'
    $answer=Read-Host 'Continuar? [s/N]'
    if ($answer -notin @('s','S','sim','SIM')) { return }
    Stop-Service $ServiceName -Force -ErrorAction SilentlyContinue
    & sc.exe delete $ServiceName | Out-Null
    Remove-Item $InstallDir -Recurse -Force -ErrorAction SilentlyContinue
    Write-Ok 'Binário e serviço removidos.'
    Write-Host "Dados preservados em $ConfigDir"
}

if ($Action -ne 'Menu') {
    switch ($Action) {
        'Install' { Install-Agent }
        'Configure' { Configure-Agent }
        'Start' { Start-Agent }
        'Doctor' { Doctor-Agent }
        'Uninstall' { Uninstall-Agent }
    }
    exit
}

while ($true) {
    Show-Environment
    Write-Host 'Escolha uma operação:' -ForegroundColor White
    Write-Host '  1) Instalação completa (recomendado)' -ForegroundColor Cyan
    Write-Host '  2) Instalar/Reparar somente o Agent'
    Write-Host '  3) Configurar/Adicionar Control Plane'
    Write-Host '  4) Diagnóstico detalhado'
    Write-Host '  5) Iniciar/Reiniciar serviço'
    Write-Host '  6) Desinstalar Agent (preserva dados)'
    Write-Host '  0) Sair'
    $choice = Read-Host 'Opção'
    try {
        switch ($choice) {
            '1' { Install-Agent; Configure-Agent; Start-Agent; Write-Ok 'Instalação concluída.'; Read-Host 'Pressione Enter' | Out-Null; break }
            '2' { Install-Agent; Read-Host 'Pressione Enter' | Out-Null }
            '3' { Configure-Agent; Read-Host 'Pressione Enter' | Out-Null }
            '4' { Doctor-Agent; Read-Host 'Pressione Enter' | Out-Null }
            '5' { Start-Agent; Read-Host 'Pressione Enter' | Out-Null }
            '6' { Uninstall-Agent; Read-Host 'Pressione Enter' | Out-Null }
            '0' { return }
            default { Write-Warn 'Opção inválida.'; Start-Sleep 1 }
        }
    } catch {
        Write-Host "✕ $($_.Exception.Message)" -ForegroundColor Red
        Read-Host 'Pressione Enter' | Out-Null
    }
}
