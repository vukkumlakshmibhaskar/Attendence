param([switch]$BackendOnly)
$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
$logs = Join-Path $root 'runtime-logs'
New-Item -ItemType Directory -Force -Path $logs | Out-Null
$python = (Get-Command python.exe).Source
& $python (Join-Path $root 'backend\setup_local_db.py')
if ($LASTEXITCODE -ne 0) { throw 'Project database startup failed.' }

function Start-ProjectService($name, $executable, $arguments, $directory, $url) {
    $pidFile = Join-Path $logs "$name.pid"
    $existing = $null
    if (Test-Path $pidFile) {
        $processId = [int](Get-Content $pidFile)
        $existing = Get-CimInstance Win32_Process -Filter "ProcessId = $processId" -ErrorAction SilentlyContinue
        if ($existing -and -not $existing.CommandLine.Contains($directory)) {
            throw "Saved $name process id belongs to a different process."
        }
    }
    if (-not $existing) {
        $uri = [uri]$url
        $client = New-Object System.Net.Sockets.TcpClient
        try {
            $client.Connect($uri.Host, $uri.Port)
            throw "Port $($uri.Port) is already occupied by another process."
        } catch [System.Net.Sockets.SocketException] {
            # Port is available.
        } finally { $client.Dispose() }
        $process = Start-Process -FilePath $executable -ArgumentList $arguments -WorkingDirectory $directory -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $logs "$name.log") -RedirectStandardError (Join-Path $logs "$name.error.log")
        $process.Id | Set-Content $pidFile
    }
    for ($attempt = 0; $attempt -lt 40; $attempt++) {
        try {
            $response = Invoke-WebRequest -UseBasicParsing -Uri $url -TimeoutSec 2
            if ($response.StatusCode -eq 200) { Write-Host "$name ready: $url"; return }
        } catch { Start-Sleep -Milliseconds 500 }
    }
    throw "$name did not become ready. Check $logs."
}

$backend = Join-Path $root 'backend'
Start-ProjectService 'backend' $python @('-u', ('"' + (Join-Path $backend 'server.py') + '"')) $backend 'http://127.0.0.1:8080/health'
if (-not $BackendOnly) {
    $frontend = Join-Path $root 'frontend'
    $node = (Get-Command node.exe).Source
    $vite = Join-Path $frontend 'node_modules\vite\bin\vite.js'
    if (-not (Test-Path $vite)) { throw 'Install frontend dependencies with npm install in frontend first.' }
    Start-ProjectService 'frontend' $node @(('"' + $vite + '"'), '--host', '127.0.0.1', '--port', '5174', '--strictPort') $frontend 'http://127.0.0.1:5174'
}
