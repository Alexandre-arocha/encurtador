param(
    [string]$ApiBaseUrl = "http://localhost:8080",
    [string]$ApiKey = "dev-api-key-change-me",
    [string]$TargetUrl = "https://example.com/?encurtador-smoke=1"
)

$ErrorActionPreference = "Stop"

function Write-Step([string]$Message) {
    Write-Host "==> $Message"
}

$ApiBaseUrl = $ApiBaseUrl.TrimEnd("/")
$headers = @{ "X-API-Key" = $ApiKey }

Write-Step "Verificando backend em $ApiBaseUrl"
$health = Invoke-RestMethod -Method Get -Uri "$ApiBaseUrl/healthz"
if ($health.status -ne "ok") {
    throw "Backend respondeu healthz, mas status nao esta ok: $($health | ConvertTo-Json -Compress)"
}

$slug = "smoke-$([DateTimeOffset]::UtcNow.ToUnixTimeSeconds())"
$body = @{
    target_url = $TargetUrl
    slug = $slug
    title = "Smoke local"
} | ConvertTo-Json

Write-Step "Criando link /$slug"
$created = Invoke-RestMethod -Method Post -Uri "$ApiBaseUrl/api/links" -Headers $headers -ContentType "application/json" -Body $body
Write-Host "Link curto: $($created.short_url)"

Write-Step "Chamando redirect sem seguir Location"
$handler = [System.Net.Http.HttpClientHandler]::new()
$handler.AllowAutoRedirect = $false
$client = [System.Net.Http.HttpClient]::new($handler)
try {
    $response = $client.GetAsync($created.short_url).GetAwaiter().GetResult()
    $statusCode = [int]$response.StatusCode
    if ($statusCode -ne 302) {
        throw "Redirect retornou HTTP $statusCode; esperado 302"
    }
    $location = $response.Headers.Location.ToString()
    if ($location -ne $TargetUrl) {
        throw "Location '$location'; esperado '$TargetUrl'"
    }
}
finally {
    $client.Dispose()
    $handler.Dispose()
}

Write-Step "Aguardando stats refletirem o clique"
$stats = $null
for ($i = 0; $i -lt 20; $i++) {
    Start-Sleep -Milliseconds 500
    $stats = Invoke-RestMethod -Method Get -Uri "$ApiBaseUrl/api/links/$($created.id)/stats?range=7d" -Headers $headers
    $dailyTotal = 0
    foreach ($point in $stats.daily) {
        $dailyTotal += [int]$point.clicks
    }
    if ([int]$stats.total_clicks -ge 1 -and $dailyTotal -ge 1) {
        Write-Step "Smoke OK"
        Write-Host "total_clicks=$($stats.total_clicks); daily_total=$dailyTotal"
        exit 0
    }
}

throw "Stats nao refletiram o clique a tempo. Ultima resposta: $($stats | ConvertTo-Json -Compress)"
