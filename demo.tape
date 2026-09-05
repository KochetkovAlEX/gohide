# Where to save the output (can be .gif, .mp4, or .webm)
Output assets/list.gif

# Set up terminal canvas size and visual settings
Set FontSize 16
Set Width 1200
Set Height 600
Set Padding 20
Set Theme "Dracula"

# Target executable name
Type "./gohide"
Enter
Sleep 1.5s

Down
Sleep 300ms
Up
Enter
Type "p"
Sleep 3.5s

# Ping one specific server node using arrow key shortcut
Down
Sleep 1s
Down
Sleep 1s
Up
Sleep 1s
Right
Sleep 1.5s

# Toggle from PROXY to TUN mode
Type "t"
Down
Sleep 1s

# Toggle back to PROXY mode
Type "t"
Sleep 800ms

# Select the server and turn the VPN connection ON
Enter
Sleep 2.5s

# Turn the connection OFF
Enter
Sleep 1.2s

# Return up to the main subscription folders view
Escape
Sleep 1s