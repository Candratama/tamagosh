# VHS tape — SSH connect demo (uses demo mode fake session)
# Generate: vhs docs/vhs/ssh.tape
# Output:   docs/vhs/ssh.gif

Output docs/vhs/ssh.gif

Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "GruvboxDark"
Set TypingSpeed 80ms
Set Padding 20

Hide
Type "tamagosh demo"
Enter
Sleep 1s
Show

# pick a connection
Sleep 1s
Down
Sleep 500ms
Down
Sleep 500ms

# connect — shows "Connecting" frame then fake shell
Enter
Sleep 4s

# inside fake shell — type a command, exit
Type "uname -a"
Sleep 1s

# Ctrl-D to exit cat (return to TUI)
Ctrl+D
Sleep 1500ms

# back to list
Type "q"
Sleep 400ms
