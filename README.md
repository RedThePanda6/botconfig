This is just a little GoLang binary that I maintain to help me configure various parts of my Twitch Stream. It started as a way to set Tags on a per-game basis and then grew from there.

It consumes a series of JSON files before spitting out a combined JSON file that is consumed by Streamer.Bot. What happens from there is a whole other series of ugliness within Streamer.Bot but that's between me and my maker.

I present this to the public in case you want to use it as inspriation for something similar yourself. Currently no license attached to it becaues I haven't given thought about which license would best apply.

You can find my stream at https://www.twitch.tv/redthepanda_. I'm almost always happy to talk about any behind the scenes stuff including going over nearly exactly how this all works.

## TODO
* Add details on filesystem/folder layout.
* Add instructions for how to read from Streamer.Bot.
  * Possibly inlcude an export of pre-built action?
* Format code with clear sections for where to add your own code.
  * Possibly break custom code out to a separate module?
