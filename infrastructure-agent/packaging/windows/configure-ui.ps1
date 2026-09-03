[CmdletBinding()]
param(
    [string]$InstallDir = "$env:ProgramFiles\InfrastructureAgent",
    [string]$ConfigDir = "$env:ProgramData\InfrastructureAgent"
)

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()

$agent = Join-Path $InstallDir 'infra-agent.exe'
$configFile = Join-Path $ConfigDir 'agent.json'
if (-not (Test-Path $agent)) {
    [System.Windows.Forms.MessageBox]::Show('O Infrastructure Agent não está instalado.', 'Infrastructure Agent', 'OK', 'Error') | Out-Null
    exit 1
}

$form = New-Object System.Windows.Forms.Form
$form.Text = 'Generic Infrastructure Agent — Configuração'
$form.Size = New-Object System.Drawing.Size(650,590)
$form.StartPosition = 'CenterScreen'
$form.FormBorderStyle = 'FixedDialog'
$form.MaximizeBox = $false
$form.BackColor = [System.Drawing.Color]::White

$title = New-Object System.Windows.Forms.Label
$title.Text = 'Generic Infrastructure Agent'
$title.Font = New-Object System.Drawing.Font('Segoe UI',16,[System.Drawing.FontStyle]::Bold)
$title.Location = New-Object System.Drawing.Point(28,20)
$title.AutoSize = $true
$form.Controls.Add($title)

$subtitle = New-Object System.Windows.Forms.Label
$subtitle.Text = 'Vincule esta máquina a um Control Plane sem expor Docker ou SSH.'
$subtitle.Font = New-Object System.Drawing.Font('Segoe UI',9)
$subtitle.ForeColor = [System.Drawing.Color]::DimGray
$subtitle.Location = New-Object System.Drawing.Point(30,55)
$subtitle.AutoSize = $true
$form.Controls.Add($subtitle)

$fields = @(
    @{ Label='Nome do Control Plane'; Name='Controller'; Default='control-plane'; Secret=$false },
    @{ Label='URL REST/HTTPS'; Name='Url'; Default='https://api.example.com'; Secret=$false },
    @{ Label='Prefixos de deployments (vírgula)'; Name='Prefixes'; Default=''; Secret=$false },
    @{ Label='Dockge API local'; Name='Dockge'; Default='http://127.0.0.1:5001'; Secret=$false },
    @{ Label='Ambiente'; Name='Environment'; Default='production'; Secret=$false },
    @{ Label='Credencial de enrollment'; Name='Enrollment'; Default=''; Secret=$true },
    @{ Label='Credencial da Dockge API'; Name='DockgeCredential'; Default=''; Secret=$true }
)
$textboxes = @{}
$y = 95
foreach ($field in $fields) {
    $label = New-Object System.Windows.Forms.Label
    $label.Text = $field.Label
    $label.Location = New-Object System.Drawing.Point(32,$y)
    $label.Size = New-Object System.Drawing.Size(220,23)
    $form.Controls.Add($label)

    $box = New-Object System.Windows.Forms.TextBox
    $box.Name = $field.Name
    $box.Text = $field.Default
    $box.Location = New-Object System.Drawing.Point(255,$y-3)
    $box.Size = New-Object System.Drawing.Size(345,26)
    if ($field.Secret) { $box.UseSystemPasswordChar = $true }
    $form.Controls.Add($box)
    $textboxes[$field.Name] = $box
    $y += 50
}

$status = New-Object System.Windows.Forms.Label
$status.Location = New-Object System.Drawing.Point(32,455)
$status.Size = New-Object System.Drawing.Size(568,45)
$status.ForeColor = [System.Drawing.Color]::DimGray
$status.Text = 'As credenciais serão gravadas em arquivos protegidos separados do agent.json.'
$form.Controls.Add($status)

$cancel = New-Object System.Windows.Forms.Button
$cancel.Text = 'Cancelar'
$cancel.Location = New-Object System.Drawing.Point(390,505)
$cancel.Size = New-Object System.Drawing.Size(100,32)
$cancel.Add_Click({ $form.Close() })
$form.Controls.Add($cancel)

$save = New-Object System.Windows.Forms.Button
$save.Text = 'Salvar e vincular'
$save.Location = New-Object System.Drawing.Point(500,505)
$save.Size = New-Object System.Drawing.Size(115,32)
$save.BackColor = [System.Drawing.Color]::FromArgb(0,120,212)
$save.ForeColor = [System.Drawing.Color]::White
$save.FlatStyle = 'Flat'
$save.Add_Click({
    try {
        if ([string]::IsNullOrWhiteSpace($textboxes.Controller.Text) -or [string]::IsNullOrWhiteSpace($textboxes.Url.Text)) {
            throw 'Nome e URL do Control Plane são obrigatórios.'
        }
        $status.ForeColor = [System.Drawing.Color]::DarkBlue
        $status.Text = 'Gravando configuração segura...'
        $form.Refresh()

        $env:INFRA_AGENT_CONTROLLER_NAME = $textboxes.Controller.Text.Trim()
        $env:INFRA_AGENT_CONTROLLER_URL = $textboxes.Url.Text.Trim()
        $env:INFRA_AGENT_ALLOWED_PREFIXES = $textboxes.Prefixes.Text.Trim()
        $env:INFRA_AGENT_DOCKGE_URL = $textboxes.Dockge.Text.Trim()
        $env:INFRA_AGENT_ENVIRONMENT = $textboxes.Environment.Text.Trim()
        $env:INFRA_AGENT_ENROLLMENT_TOKEN = $textboxes.Enrollment.Text
        $env:INFRA_AGENT_DOCKGE_TOKEN = $textboxes.DockgeCredential.Text
        $env:INFRA_AGENT_DATA_DIR = Join-Path $ConfigDir 'data'

        & $agent --config $configFile configure
        if ($LASTEXITCODE -ne 0) { throw 'O Agent recusou a configuração.' }

        if (-not [string]::IsNullOrWhiteSpace($textboxes.Enrollment.Text)) {
            $status.Text = 'Configuração salva. Realizando enrollment...'
            $form.Refresh()
            & $agent --config $configFile enroll
            if ($LASTEXITCODE -ne 0) { throw 'A configuração foi salva, mas o enrollment não foi concluído.' }
        }

        $service = Get-Service -Name InfrastructureAgent -ErrorAction SilentlyContinue
        if ($service) { Restart-Service InfrastructureAgent -Force -ErrorAction SilentlyContinue }
        $status.ForeColor = [System.Drawing.Color]::DarkGreen
        $status.Text = 'Máquina configurada e vinculada com sucesso.'
        [System.Windows.Forms.MessageBox]::Show('Configuração concluída com sucesso.', 'Infrastructure Agent', 'OK', 'Information') | Out-Null
    } catch {
        $status.ForeColor = [System.Drawing.Color]::DarkRed
        $status.Text = $_.Exception.Message
        [System.Windows.Forms.MessageBox]::Show($_.Exception.Message, 'Infrastructure Agent', 'OK', 'Error') | Out-Null
    } finally {
        'INFRA_AGENT_CONTROLLER_NAME','INFRA_AGENT_CONTROLLER_URL','INFRA_AGENT_ALLOWED_PREFIXES','INFRA_AGENT_DOCKGE_URL','INFRA_AGENT_ENVIRONMENT','INFRA_AGENT_ENROLLMENT_TOKEN','INFRA_AGENT_DOCKGE_TOKEN','INFRA_AGENT_DATA_DIR' | ForEach-Object { Remove-Item "Env:$_" -ErrorAction SilentlyContinue }
    }
})
$form.Controls.Add($save)
$form.AcceptButton = $save
$form.CancelButton = $cancel
[void]$form.ShowDialog()
