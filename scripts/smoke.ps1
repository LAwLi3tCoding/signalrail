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
    $testHome = Join-Path $tmp "home"
    New-Item -ItemType Directory -Force -Path (Join-Path $project ".git") | Out-Null
    "2`ny`n" | & $binary config --lang en --scope project --project $project --home $testHome | Out-Null
    Assert-NativeSuccess "config"
    $configText = @(
        'version = 1',
        'segments = ["model", "context"]',
        '',
        '[runtime.codex]',
        'segments = ["model", "context"]'
    ) -join [Environment]::NewLine
    Set-Content -Path (Join-Path $project ".signalrail.toml") -Value $configText -NoNewline
    $effectivePreview = & $binary preview --project $project --home $testHome
    Assert-NativeSuccess "effective preview"
    if ($effectivePreview -notmatch "GPT-5\.5") { throw "effective preview did not include configured model segment" }
    if ($effectivePreview -notmatch "CTX 38% left") { throw "effective preview did not include configured context segment" }
    if ($effectivePreview -match "Build renderer") { throw "effective preview ignored configured segments" }
    $codexDryRun = (& $binary install codex --scope project --dry-run --home $testHome --project $project) -join [Environment]::NewLine
    Assert-NativeSuccess "codex dry-run"
    if ($codexDryRun -notmatch 'status_line = \["model-with-reasoning", "context-remaining"\]') {
        throw "Codex dry-run did not compile configured runtime segments"
    }
    if ($codexDryRun -match "git-branch") { throw "Codex dry-run ignored configured runtime segments" }
    & $binary task set "Smoke task" --total 2 --project $project | Out-Null
    Assert-NativeSuccess "task set"
    & $binary task step --project $project | Out-Null
    Assert-NativeSuccess "task step"
    $task = & $binary task show --project $project | ConvertFrom-Json
    Assert-NativeSuccess "task show"
    if ($task.Step -ne 1) { throw "task step mismatch" }

    & $binary install claude --scope project --home $testHome --project $project | Out-Null
    Assert-NativeSuccess "install claude"
    & $binary install codex --scope project --home $testHome --project $project | Out-Null
    Assert-NativeSuccess "install codex"
    if ((Get-Content (Join-Path $project ".claude/settings.json") -Raw) -notmatch "signalrail render") {
        throw "Claude status line was not installed"
    }
    if ((Get-Content (Join-Path $project ".codex/config.toml") -Raw) -notmatch "SignalRail managed status line") {
        throw "Codex status line was not installed"
    }
    & $binary doctor --json --home $testHome --project $project | ConvertFrom-Json | Out-Null
    Assert-NativeSuccess "doctor"
    $rendered = Get-Content testdata/claude/full.json | & $binary render --runtime claude --width 80 --project $project --home $testHome
    Assert-NativeSuccess "render"
    if ($rendered -notmatch "Opus 4\.7") { throw "rendered model was not found" }
    Write-Output "SignalRail smoke checks passed"
}
finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
