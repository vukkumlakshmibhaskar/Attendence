$ErrorActionPreference = 'Stop'
$root = $PSScriptRoot
foreach ($name in @('frontend', 'backend')) {
    $pidFile = Join-Path $root "runtime-logs\$name.pid"
    if (Test-Path $pidFile) {
        $processId = [int](Get-Content $pidFile)
        $process = Get-CimInstance Win32_Process -Filter "ProcessId = $processId" -ErrorAction SilentlyContinue
        if ($process) {
            if (-not $process.CommandLine.Contains((Join-Path $root $name))) {
                throw "Refusing to stop process $processId because it is not this project's $name."
            }
            Stop-Process -Id $processId
        }
        Remove-Item -LiteralPath $pidFile
    }
}
& python.exe (Join-Path $root 'backend\setup_local_db.py') --stop
if ($LASTEXITCODE -ne 0) { throw 'Database stop failed; see PostgreSQL log.' }
