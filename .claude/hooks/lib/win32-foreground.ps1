# win32-foreground.ps1 — Force terminal to foreground (nuclear mode)
# Disable Windows foreground lock, steal focus, play sound, flash taskbar
# Supports multiple terminal windows: enumerates all windows and matches by title hint

param([string]$Hint = "")

$LogFile = Join-Path $PSScriptRoot "win32-foreground.log"

function Log-Msg {
    param([string]$Message)
    $ts = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Add-Content -Path $LogFile -Value "[$ts] $Message"
}

Log-Msg "=== win32-foreground started (hint='$Hint') ==="

Add-Type @"
using System;
using System.Runtime.InteropServices;
using System.Text;
public class WinAPI {
    [DllImport("user32.dll")]
    public static extern IntPtr GetForegroundWindow();
    [DllImport("user32.dll")]
    public static extern bool SetForegroundWindow(IntPtr hWnd);
    [DllImport("user32.dll")]
    public static extern bool BringWindowToTop(IntPtr hWnd);
    [DllImport("user32.dll")]
    public static extern bool ShowWindow(IntPtr hWnd, int nCmdShow);
    [DllImport("user32.dll")]
    public static extern bool IsIconic(IntPtr hWnd);
    [DllImport("user32.dll")]
    public static extern void keybd_event(byte bVk, byte bScan, uint dwFlags, UIntPtr dwExtraInfo);
    [DllImport("user32.dll")]
    public static extern bool SetWindowPos(IntPtr hWnd, IntPtr hWndInsertAfter, int X, int Y, int cx, int cy, uint uFlags);
    [DllImport("user32.dll")]
    public static extern bool FlashWindow(IntPtr hWnd, bool bInvert);
    [DllImport("user32.dll")]
    public static extern bool SystemParametersInfo(uint uiAction, uint uiParam, IntPtr pvParam, uint fWinIni);
    public static readonly IntPtr HWND_TOPMOST = new IntPtr(-1);
    public static readonly IntPtr HWND_NOTOPMOST = new IntPtr(-2);
    public const uint SWP_NOSIZE = 0x0001;
    public const uint SWP_NOMOVE = 0x0002;
    public const int SW_RESTORE = 9;
    public const int SW_SHOW = 5;
    public const uint SPI_GETFOREGROUNDLOCKTIMEOUT = 0x2000;
    public const uint SPI_SETFOREGROUNDLOCKTIMEOUT = 0x2001;
    public const uint SPIF_SENDCHANGE = 0x0002;
    public delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);
    [DllImport("user32.dll")]
    public static extern bool EnumWindows(EnumWindowsProc lpEnumFunc, IntPtr lParam);
    [DllImport("user32.dll")]
    public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint lpdwProcessId);
    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    public static extern int GetWindowText(IntPtr hWnd, StringBuilder lpString, int nMaxCount);
    [DllImport("user32.dll")]
    public static extern bool IsWindowVisible(IntPtr hWnd);
}
"@

# --- Enumerate visible windows for a specific PID ---
function Get-WindowsForPid {
    param([int]$TargetPid)
    $script:enumResults = @()
    $callback = [WinAPI+EnumWindowsProc]{
        param([IntPtr]$hWnd, [IntPtr]$lParam)
        $procId = [uint32]0
        [WinAPI]::GetWindowThreadProcessId($hWnd, [ref]$procId) | Out-Null
        if ($procId -eq $TargetPid -and [WinAPI]::IsWindowVisible($hWnd)) {
            $sb = New-Object System.Text.StringBuilder 256
            [WinAPI]::GetWindowText($hWnd, $sb, 256) | Out-Null
            $title = $sb.ToString()
            if ($title) {
                $script:enumResults += @{ Hwnd = $hWnd; Title = $title }
            }
        }
        return $true
    }
    [WinAPI]::EnumWindows($callback, [IntPtr]::Zero) | Out-Null
    return $script:enumResults
}

# --- Find terminal window ---
function Find-TerminalWindow {
    param([string]$TitleHint = "")
    $terminalNames = @(
        'WindowsTerminal', 'WindowsTerminal.exe', 'wt', 'wt.exe',
        'ConEmu64', 'ConEmu64.exe', 'cmder', 'cmder.exe'
    )

    # Walk parent chain to find terminal PID
    $currentPid = $PID
    $seen = @{}
    $ancestors = @()
    for ($i = 0; $i -lt 20; $i++) {
        try {
            $proc = Get-CimInstance Win32_Process -Filter "ProcessId = $currentPid" -ErrorAction SilentlyContinue
            if (-not $proc) { break }
            $parentPid = $proc.ParentProcessId
            if ($seen.ContainsKey($parentPid) -or $parentPid -eq $currentPid) { break }
            $seen[$parentPid] = $true
            $parentProc = Get-CimInstance Win32_Process -Filter "ProcessId = $parentPid" -ErrorAction SilentlyContinue
            if ($parentProc) { $ancestors += @{ Pid = $parentPid; Name = $parentProc.Name } }
            $currentPid = $parentPid
        } catch { break }
    }

    # Strategy 1: Find terminal in ancestors, enumerate its windows
    foreach ($anc in $ancestors) {
        if ($terminalNames -contains $anc.Name) {
            $windows = Get-WindowsForPid -TargetPid $anc.Pid
            if ($windows.Count -gt 0) {
                $picked = Pick-Window -Windows $windows -Hint $TitleHint
                return @{ Pid = $anc.Pid; Name = $anc.Name; Hwnd = $picked.Hwnd; Title = $picked.Title }
            }
            # No visible windows via EnumWindows, try MainWindowHandle
            try {
                $p = [System.Diagnostics.Process]::GetProcessById($anc.Pid)
                if ($p.MainWindowHandle -ne [IntPtr]::Zero) {
                    return @{ Pid = $anc.Pid; Name = $anc.Name; Hwnd = $p.MainWindowHandle; Title = $p.MainWindowTitle }
                }
            } catch {}
        }
    }

    # Strategy 2: Any ancestor with a window
    foreach ($anc in $ancestors) {
        try {
            $p = [System.Diagnostics.Process]::GetProcessById($anc.Pid)
            if ($p.MainWindowHandle -ne [IntPtr]::Zero) {
                return @{ Pid = $anc.Pid; Name = $anc.Name; Hwnd = $p.MainWindowHandle; Title = $p.MainWindowTitle }
            }
        } catch {}
    }

    # Strategy 3: Search by process name (last resort)
    foreach ($name in @('WindowsTerminal', 'ConEmu64', 'cmder')) {
        try {
            $procs = [System.Diagnostics.Process]::GetProcessesByName($name)
            foreach ($p in $procs) {
                if ($p.MainWindowHandle -ne [IntPtr]::Zero) {
                    return @{ Pid = $p.Id; Name = $name; Hwnd = $p.MainWindowHandle; Title = $p.MainWindowTitle }
                }
            }
        } catch {}
    }
    return $null
}

# --- Pick best window from candidates ---
function Pick-Window {
    param([array]$Windows, [string]$Hint = "")
    if ($Windows.Count -eq 1) { return $Windows[0] }

    Log-Msg "Multiple windows found ($($Windows.Count)), selecting best match..."

    # Priority 1: exact hint match
    if ($Hint) {
        foreach ($w in $Windows) {
            if ($w.Title -like "*$Hint*") {
                Log-Msg "Hint '$Hint' matched: $($w.Title)"
                return $w
            }
        }
    }

    # Priority 2: Claude Code indicators
    # ● (U+25CF) appears as non-ASCII, so check first char codepoint > 127
    # Also match "claude" keyword case-insensitively
    foreach ($w in $Windows) {
        $firstCharCode = if ($w.Title.Length -gt 0) { [int]$w.Title[0] } else { 0 }
        if ($firstCharCode -gt 127 -or $w.Title.ToLower().Contains('claude')) {
            Log-Msg "Claude indicator matched (char=$firstCharCode): $($w.Title)"
            return $w
        }
    }

    # Priority 3: fall back to first window
    Log-Msg "No hint or Claude match, using first window: $($Windows[0].Title)"
    return $Windows[0]
}

# --- Main ---
$info = Find-TerminalWindow -TitleHint $Hint
if (-not $info) {
    Log-Msg "ERROR: No terminal found"
    exit 1
}
$hwnd = $info.Hwnd
Log-Msg "Target: $($info.Name) PID=$($info.Pid) hwnd=$hwnd title='$($info.Title)'"

# Step 1: Save and disable foreground lock timeout
$oldTimeout = [IntPtr]::Zero
[WinAPI]::SystemParametersInfo([WinAPI]::SPI_GETFOREGROUNDLOCKTIMEOUT, 0, $oldTimeout, 0) | Out-Null
Log-Msg "Current foreground lock timeout: $oldTimeout"

$setResult = [WinAPI]::SystemParametersInfo(
    [WinAPI]::SPI_SETFOREGROUNDLOCKTIMEOUT, 0,
    [IntPtr]::Zero, [WinAPI]::SPIF_SENDCHANGE)
Log-Msg "Disable foreground lock: $setResult"

# Step 2: Restore if minimized
if ([WinAPI]::IsIconic($hwnd)) {
    [WinAPI]::ShowWindow($hwnd, [WinAPI]::SW_RESTORE)
    Start-Sleep -Milliseconds 100
}
[WinAPI]::ShowWindow($hwnd, [WinAPI]::SW_SHOW)

# Step 3: Alt key trick
[WinAPI]::keybd_event(0x12, 0, 0, [UIntPtr]::Zero)
[WinAPI]::keybd_event(0x12, 0, 0x2, [UIntPtr]::Zero)

# Step 4: COM AppActivate
try {
    $wsh = New-Object -ComObject WScript.Shell
    if ($info.Title) {
        $r = $wsh.AppActivate($info.Title)
        Log-Msg "COM AppActivate(title)=$r"
    }
    if (-not $r) {
        $r = $wsh.AppActivate($info.Name)
        Log-Msg "COM AppActivate(name)=$r"
    }
} catch {
    Log-Msg "COM failed"
}

# Step 5: SetForegroundWindow (should work now with lock disabled)
$r1 = [WinAPI]::SetForegroundWindow($hwnd)
$r2 = [WinAPI]::BringWindowToTop($hwnd)
Log-Msg "SetForeground=$r1 BringToTop=$r2"

# Step 6: Verify
Start-Sleep -Milliseconds 100
$nowFg = [WinAPI]::GetForegroundWindow()
if ($nowFg -eq $hwnd) {
    Log-Msg "SUCCESS: Window is now foreground!"
} else {
    Log-Msg "WARNING: Foreground is still $nowFg (target $hwnd)"
}

# Step 7: TOPMOST for 1 second to ensure visibility
[WinAPI]::SetWindowPos($hwnd, [WinAPI]::HWND_TOPMOST, 0, 0, 0, 0,
    [WinAPI]::SWP_NOSIZE -bor [WinAPI]::SWP_NOMOVE)
Start-Sleep -Milliseconds 1000
[WinAPI]::SetWindowPos($hwnd, [WinAPI]::HWND_NOTOPMOST, 0, 0, 0, 0,
    [WinAPI]::SWP_NOSIZE -bor [WinAPI]::SWP_NOMOVE)

# Step 8: Flash taskbar
for ($i = 0; $i -lt 6; $i++) {
    [WinAPI]::FlashWindow($hwnd, $true)
    Start-Sleep -Milliseconds 200
}

# Step 9: Play sound
try { [System.Media.SystemSounds]::Asterisk.Play() } catch {}

# Step 10: Restore original foreground lock timeout
[WinAPI]::SystemParametersInfo(
    [WinAPI]::SPI_SETFOREGROUNDLOCKTIMEOUT, 0,
    $oldTimeout, [WinAPI]::SPIF_SENDCHANGE) | Out-Null
Log-Msg "Foreground lock timeout restored to $oldTimeout"

Log-Msg "=== done ==="
