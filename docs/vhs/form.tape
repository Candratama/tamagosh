# VHS tape — add new connection form
# Generate: vhs docs/vhs/form.tape
# Output:   docs/vhs/form.gif

Output docs/vhs/form.gif

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

# open new connection form
Sleep 1s
Type "n"
Sleep 1s

# fill in fields
Type "demo"
Tab
Sleep 400ms
Type "example.com"
Tab
Sleep 400ms
Type "22"
Tab
Sleep 400ms
Type "root"
Tab
Sleep 400ms
Type "secret"
Sleep 1s

# cancel (we don't actually want to save a demo)
Escape
Sleep 800ms

# quit
Type "q"
Sleep 400ms
