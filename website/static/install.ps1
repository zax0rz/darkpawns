# Stable entry point at darkpawns.org/install.ps1. Compatible with Windows PowerShell 5.1+.
# The publisher key below matches website/static/install.sh and changes only with a
# deliberate stub update on key rotation. Every release publishes install.ps1.sig
# beside install.ps1, and the installer runs only after that signature verifies.
# Keep the complete function defined before its single invocation so an interrupted
# download cannot execute a partial body.
function Invoke-DarkPawnsInstall {
    $ErrorActionPreference = 'Stop'
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    $base = 'https://github.com/zax0rz/darkpawns/releases/latest/download'
    $work = Join-Path ([IO.Path]::GetTempPath()) ('darkpawns-bootstrap-' + [Guid]::NewGuid().ToString('N'))
    [void][IO.Directory]::CreateDirectory($work)

    function Fetch-SignedAsset([string]$Name, [long]$MaxBytes) {
        $dest = Join-Path $work $Name
        Add-Type -AssemblyName System.Net.Http
        $handler = New-Object Net.Http.HttpClientHandler
        $handler.AllowAutoRedirect = $false
        $client = New-Object Net.Http.HttpClient($handler)
        $client.Timeout = [TimeSpan]::FromSeconds(180)
        $uri = [Uri]($base + '/' + $Name)
        try {
            for ($redirects = 0; $redirects -lt 6; $redirects++) {
                if ($uri.Scheme -ne 'https' -or $uri.UserInfo) { throw 'Unsafe download redirect.' }
                $response = $client.GetAsync($uri, [Net.Http.HttpCompletionOption]::ResponseHeadersRead).GetAwaiter().GetResult()
                try {
                    $code = [int]$response.StatusCode
                    if ($code -ge 300 -and $code -lt 400) {
                        $uri = New-Object Uri($uri, $response.Headers.Location)
                        continue
                    }
                    if ($code -ne 200) { throw ('Download returned HTTP ' + $code) }
                    $inStream = $response.Content.ReadAsStreamAsync().GetAwaiter().GetResult()
                    $outStream = [IO.File]::Create($dest)
                    try {
                        $buffer = New-Object byte[] 65536
                        [long]$total = 0
                        while (($read = $inStream.Read($buffer, 0, $buffer.Length)) -gt 0) {
                            $total += $read
                            if ($total -gt $MaxBytes) { throw 'Installer download exceeds size limit.' }
                            $outStream.Write($buffer, 0, $read)
                        }
                    } finally {
                        $outStream.Dispose()
                        $inStream.Dispose()
                    }
                    return $dest
                } finally {
                    $response.Dispose()
                }
            }
            throw 'Too many download redirects.'
        } finally {
            $client.Dispose()
            $handler.Dispose()
        }
    }

    try {
        $installer = Fetch-SignedAsset 'install.ps1' 1048576
        $signature = Fetch-SignedAsset 'install.ps1.sig' 1024
        $rsa = New-Object Security.Cryptography.RSACryptoServiceProvider
        $rsa.PersistKeyInCsp = $false
        $rsa.FromXmlString('<RSAKeyValue><Modulus>yFZWeEzNp0iaUB3CZtI0N3uEelzM9QYWmsqfKf9Uo/Q5LfCtM/kwE4dDKaFYZ7AnY8gp3CquBUsi4K2+Om+ZRHeTAgwbjRfBDP+Z7dFHrvTh9l9bH/gxX6Qv7W0Rq2PcogtCmZTJdOmFSVMQ8VnAzEVo5R5ZeQoyfuwR9a2W8MZ5r91lsfebSTz+oxSmmcwVDNhq3M4nV5mIaEwvnxS9lPvqj0zdf0L8Q5AQxBd4G7t4Gt7ipp+7h/JrQ6wSOZvcNC+ly6fIMuswp7sAwwrqwawhnX+s/Ucw148NepkiMlZ6kCF3qgNfDfPrtEkZvDSjHN36M9JQErghxCW1rgpkbi38qr3M8zm74RSM4imW4l0cnISOKusk6OWhAuTJOXPgckg7tQ6l/LgbbBI7eEBxORwBDCCZHSt9yNc58OzIt2wvzvDlQ0mK/RSh1MFpPYc707dk5JZILMRYL0jJbr2sdLZWgNMrcaDmvHTp2ckSOIJI5Cq02NHCnrm5LiO2IRQF</Modulus><Exponent>AQAB</Exponent></RSAKeyValue>')
        try {
            $ok = $rsa.VerifyData(
                [IO.File]::ReadAllBytes($installer),
                'SHA256',
                [IO.File]::ReadAllBytes($signature)
            )
            if (-not $ok) { throw 'Publisher verification failed. Nothing ran.' }
        } finally {
            $rsa.Dispose()
        }
        & powershell.exe -NoProfile -ExecutionPolicy Bypass -File $installer
        if ($LASTEXITCODE -ne 0) { throw 'Installer did not complete.' }
    } finally {
        Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue
    }
}
Invoke-DarkPawnsInstall
