# parallel-dbus

Implement basic cli client that communicates with NetworkManage  trough it's DBUS interface
The client should have the following functionality(subcommands)
- scan - return available networks by SSID  and signal strength
- connect SSID - connect to SSID using the password read from stdin
- disconnect - disconnect from currently connected network
- Adapter name could also be an argument or ENV variable
