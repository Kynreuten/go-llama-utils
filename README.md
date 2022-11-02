# go-llama-utils
Go Language Utilities. Mostly for my learning, it's possible some will be practical



# Using Caller on a Mac
You must put your `logstash.conf` file into LogStash's `./config/` folder for this to work. It seems to ignore absolute paths?! I have no idea why the two layers of ../ are required. Only one and it sees a space in the config path still...
```
/path/to/go/bin/caller --debug.main --cmd /path/to/logstash-8.4.2/bin/logstash --cmdArg f=../../config/my.conf /path/to/my.mac.env /path/to/my.local.env
```