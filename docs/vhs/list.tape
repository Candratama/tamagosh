# VHS tape — connection list view
# Generate: vhs docs/vhs/list.tape
# Output:   docs/vhs/list.gif

Output docs/vhs/list.gif

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

# show the connection list
Sleep 2s

# move down through entries
Down
Sleep 600ms
Down
Sleep 600ms
Down
Sleep 600ms
Up
Sleep 800ms

# briefly open filter
Type "/"
Sleep 400ms
Type "at"
Sleep 1s
Escape
Sleep 800ms

# quit
Type "q"
Sleep 500ms
