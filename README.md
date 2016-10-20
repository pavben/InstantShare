# InstantShare [![Build Status](https://travis-ci.org/pavben/InstantShare.svg?branch=master)](https://travis-ci.org/pavben/InstantShare) [![GoDoc](https://godoc.org/github.com/pavben/InstantShare?status.svg)](https://godoc.org/github.com/pavben/InstantShare)

Instantly share images, videos, and other files on the web.

Features
--------

-	**Instant links**
	-	When you share via Instant Share, you get a shareable link in your clipboard right away, so you can paste it in a conversation, in a forum post, or anywhere. That link is usable right away because Instant Share streams both the uploads and downloads in the background, allowing downloads to begin before you finish uploading.
-	**Share any file type**
	-	You can share any image, video or other file types, including .html, .css, .txt and get a direct link to it. If possible, the files will be displayed in browser. Most browsers support streaming video downloads, making Instant Share the quickest way to share a video.

Instant Share consists of a server and client.

The client runs in your operating system's tray bar for quick access. The server is responsible for accepting uploads and serving those files.

Platforms
---------

Instant Share client is fully supported on macOS. Linux and Windows clients are possible pending support being implemented in [`trayhost`](https://github.com/shurcooL/trayhost) package that Instant Share client uses for creating a tray icon.

Instant Share server runs on macOS, Linux, Windows or any other platform that Go supports.

Screenshots
-----------

![Screenshot 1](https://cloud.githubusercontent.com/assets/1924134/8891878/8dd7a024-32ee-11e5-8eca-1994f8503094.png)

![Screenshot 2](https://cloud.githubusercontent.com/assets/1924134/8891877/8dc67272-32ee-11e5-9656-7f9749394ad3.png)

License
-------

-	[MIT License](LICENSE)
