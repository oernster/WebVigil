# The WebVigil gate. build.ps1 runs it first and stops on any failure; there is no switch to skip it.
#
#   ./test.ps1
#
# In order: gofmt, go vet, staticcheck, the Go suite with the coverage floor on the domain and
# application layers, then the front end's type check, lint and tests. Read the exit code: 0 means
# every step passed.
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

# The floor: domain and application are reachable with no network, disk, clock or desktop, so
# anything short of 100% there is a decision nobody made. Infrastructure and the UI are tested but
# not floored; they need a real desktop for their Windows halves.
$coverageFloor = 100
$flooredPackages = @('./internal/domain/', './internal/application/')

function Step($name, [scriptblock]$body) {
    Write-Host "== $name"
    & $body
    if ($LASTEXITCODE -ne 0) { throw "$name failed with exit code $LASTEXITCODE" }
}

Step 'gofmt' {
    # Every root .go file by pattern, so a new facade file or test is checked without being listed;
    # the folders by name, which keeps frontend/node_modules (it ships Go of its own) out.
    $rootGo = (Get-ChildItem -Path $root -Filter '*.go' -File).FullName
    $unformatted = gofmt -l internal tests tools installer $rootGo
    if ($unformatted) { Write-Host $unformatted; $global:LASTEXITCODE = 1 } else { $global:LASTEXITCODE = 0 }
}
Step 'go vet' { go vet ./... }
Step 'staticcheck' { go run honnef.co/go/tools/cmd/staticcheck@latest ./... }
Step 'go test' { go test ./... -count=1 }

foreach ($pkg in $flooredPackages) {
    $profile = Join-Path $env:TEMP ('webvigil-cover-' + ($pkg -replace '[^a-z]', '') + '.out')
    Step "coverage $pkg" { go test $pkg -count=1 -coverprofile="$profile" | Out-Null }
    $total = (go tool cover -func="$profile" | Select-String 'total:').ToString().Split()[-1].TrimEnd('%')
    if ([double]$total -lt $coverageFloor) {
        go tool cover -func="$profile" | Select-String -NotMatch '100.0%'
        throw "coverage of $pkg is $total%, below the floor of $coverageFloor%"
    }
    Write-Host "   $pkg $total%"
}

Push-Location (Join-Path $root 'frontend')
try {
    Step 'tsc' { npx tsc --noEmit }
    Step 'eslint' { npx eslint src }
    Step 'vitest' { npx vitest run }
} finally {
    Pop-Location
}
Write-Host 'Gate passed.'
