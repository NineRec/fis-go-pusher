# Fis-Go-Pusher

Using golang to implement a fis3-http-pusher compatable program.

### Background

I learned from a FE developer using fis3 to push code from local machine to remote machine.

But it's really resource expansive for the fis3 to keep running om my Mac.

This code use `fsnotify/fsnotify` to monitor file system event and push the file to remote machine using fis3's `reciever.php`;

### Usage

```
fis-go-pusher -a=proj-name -c=/path/to/conf.json
```
