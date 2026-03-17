# VHS tape file for recording bfr demo
# Run: vhs demo.tape
# Requires: brew install vhs

Output assets/demo.gif

Set FontSize 16
Set Width 1200
Set Height 600
Set Theme "Dracula"
Set TypingSpeed 80ms

# Launch bfr
Type "bfr"
Enter
Sleep 3s

# Navigate file list
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms
Type "j"
Sleep 600ms

# Switch to file pane
Tab
Sleep 1s

# Scroll through blocks
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms
Type "j"
Sleep 500ms

# Increase indent level
Type "l"
Sleep 500ms
Type "l"
Sleep 500ms

# Mark block as reviewed
Space
Sleep 1s

# Navigate and review another block
Type "j"
Sleep 500ms
Space
Sleep 1s

# Switch back to file list
Tab
Sleep 1s

# Show updated percentages
Sleep 2s

# Quit
Type "q"
