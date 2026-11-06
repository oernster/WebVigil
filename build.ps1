# Builds WebVigil: the gate, the icons, the application and the setup program.
#
#   ./build.ps1                 everything
#   ./build.ps1 -SkipInstaller  stop after WebVigil.exe
#
# Outputs:
#   build/bin/WebVigil.exe                 the application
#   dist-installer/WebVigilSetup.exe       the setup program, carrying the application as its payload
param(
    [switch]$SkipInstaller
)

$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

$version = (Get-Content (Join-Path $root 'VERSION')).Trim()
Write-Host "Building WebVigil $version"

# The repository that publishes the releases, read from this checkout's own git remote so the
# source names no account. The update check asks its latest release; without one it reports
# every check as unreachable rather than asking the wrong project.
$releaseRepo = ''
$origin = git remote get-url origin 2>$null
if ($LASTEXITCODE -eq 0 -and $origin -match 'github\.com[:/]([^/]+/[^/]+?)(\.git)?$') {
    $releaseRepo = $Matches[1]
    Write-Host "Release repository: $releaseRepo"
} else {
    Write-Warning 'No GitHub origin remote: this build cannot check for updates.'
}

# 1. Every icon is derived from the supplied masters in assets/. This comes before the gate because
#    the front end's tests import the generated icons: a new master has no icon until this runs.
go run ./tools/genicons
if ($LASTEXITCODE -ne 0) { throw "genicons failed with exit code $LASTEXITCODE" }

# 2. The gate runs before anything is built and cannot be skipped.
& (Join-Path $root 'test.ps1')
if ($LASTEXITCODE -ne 0) { throw "the gate failed with exit code $LASTEXITCODE" }

# 3. The application, with VERSION stamped into main.appVersion and the release repository into
#    main.releaseRepo.
wails build -ldflags "-X main.appVersion=$version -X main.releaseRepo=$releaseRepo"
if ($LASTEXITCODE -ne 0) { throw "wails build failed with exit code $LASTEXITCODE" }

if ($SkipInstaller) {
    Write-Host "Done: build/bin/WebVigil.exe ($version)"
    exit 0
}

# 4. The payload: the application and its licence, which setup shows from inside the payload.
$payload = Join-Path $root 'installer/payload.zip'
$staging = Join-Path $env:TEMP 'webvigil-payload'
if (Test-Path $staging) { Remove-Item $staging -Recurse -Force }
New-Item -ItemType Directory -Path $staging | Out-Null
Copy-Item (Join-Path $root 'build/bin/WebVigil.exe') $staging
Copy-Item (Join-Path $root 'LICENSE') $staging
if (Test-Path $payload) { Remove-Item $payload }
Compress-Archive -Path (Join-Path $staging '*') -DestinationPath $payload

try {
    # 5. The setup page's licence reads itself with the application's own scroller: the page has no
    #    build step, so the one implementation is bundled into a plain script beside it.
    $scrollBundle = Join-Path $root 'installer/frontend/dist/autoscroll.js'
    & (Join-Path $root 'frontend/node_modules/.bin/esbuild.cmd') (Join-Path $root 'frontend/src/autoScrollDom.ts') --bundle --format=iife --global-name=WebVigilScroll "--outfile=$scrollBundle"
    if ($LASTEXITCODE -ne 0) { throw "bundling the setup page's scroller failed with exit code $LASTEXITCODE" }

    # 6. The setup program.
    Push-Location (Join-Path $root 'installer')
    try {
        wails build -ldflags "-X main.appVersion=$version"
        if ($LASTEXITCODE -ne 0) { throw "setup build failed with exit code $LASTEXITCODE" }
    } finally {
        Pop-Location
    }
    $dist = Join-Path $root 'dist-installer'
    New-Item -ItemType Directory -Force -Path $dist | Out-Null
    Copy-Item (Join-Path $root 'installer/build/bin/WebVigilSetup.exe') (Join-Path $dist 'WebVigilSetup.exe') -Force
} finally {
    # 7. Restore the committed empty-zip placeholder so the tree builds without a full build.
    $placeholder = New-Object byte[] 22
    $placeholder[0] = 0x50; $placeholder[1] = 0x4B; $placeholder[2] = 0x05; $placeholder[3] = 0x06
    [System.IO.File]::WriteAllBytes($payload, $placeholder)
}

Write-Host "Done: dist-installer/WebVigilSetup.exe ($version)"
