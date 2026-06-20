$ErrorActionPreference = "Stop"

function Assert-NativeSuccess([string]$Step) {
    if ($LASTEXITCODE -ne 0) { throw "$Step failed with exit code $LASTEXITCODE" }
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("signalrail-" + [guid]::NewGuid())

try {
    New-Item -ItemType Directory -Force -Path $tmp | Out-Null
    Set-Location $repoRoot
    $binary = Join-Path $tmp "signalrail.exe"
    go build -o $binary ./cmd/signalrail
    Assert-NativeSuccess "build"

    foreach ($preset in @("wide", "standard", "compact", "minimal")) {
        $output = & $binary preview --preset $preset
        Assert-NativeSuccess "preview $preset"
        if ([string]::IsNullOrWhiteSpace($output)) { throw "empty $preset preview" }
    }

    $project = Join-Path $tmp "project"
    $home = Join-Path $tmp "home"
    New-Item -ItemType Directory -Force -Path (Join-Path $project ".git") | Out-Null
    "2`ny`n" | & $binary config --lang en --scope project --project $project --home $home | Out-Null
    Assert-NativeSuccess "config"
    & $binary task set "Smoke task" --total 2 --project $project | Out-Null
    Assert-NativeSuccess "task set"
    & $binary task step --project $project | Out-Null
    Assert-NativeSuccess "task step"
    $task = & $binary task show --project $project | ConvertFrom-Json
    Assert-NativeSuccess "task show"
    if ($task.Step -ne 1) { throw "task step mismatch" }

    & $binary install claude --scope project --home $home --project $project | Out-Null
    Assert-NativeSuccess "install claude"
    & $binary install codex --scope project --home $home --project $project | Out-Null
    Assert-NativeSuccess "install codex"
    if ((Get-Content (Join-Path $project ".claude/settings.json") -Raw) -notmatch "signalrail render") {
        throw "Claude status line was not installed"
    }
    if ((Get-Content (Join-Path $project ".codex/config.toml") -Raw) -notmatch "SignalRail managed status line") {
        throw "Codex status line was not installed"
    }
    & $binary doctor --json --home $home --project $project | ConvertFrom-Json | Out-Null
    Assert-NativeSuccess "doctor"
    $rendered = Get-Content testdata/claude/full.json | & $binary render --runtime claude --width 80 --project $project --home $home
    Assert-NativeSuccess "render"
    if ($rendered -notmatch "Opus 4\.7") { throw "rendered model was not found" }
    Write-Output "SignalRail smoke checks passed"
}
finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
