# Minecraft Updater
Simple tool to help keep minecraft's server up-to-date

[![Build Status](https://travis-ci.org/joekr/minecraft_updater.svg?branch=master)](https://travis-ci.org/joekr/minecraft_updater)

### Flags
You can change many of the defaults using the following flags:

* updateInterval - Integer value in hours to check for updates (default: 4)
* debug - Shows the debug logs (default: false)
* releaseOnly - Only download releases (default: false)
* serverPath - Where the server is downloaded and run from (default: current ".")
* worldDir - World dir location (default: current ".")
* backupDir - Backup dir location (default: current ".")
* downloadURL - The url used to download server/version info (default: "https://s3.amazonaws.com/Minecraft.Download/versions")

### Example how to run

`minecraft_updater_osx --debug=true`

### First time run
If this is the first time running you will need to agree to the eula and re-run the updater.
