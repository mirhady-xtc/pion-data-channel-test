
let HOST = "54.66.197.78" 
let ws = new WebSocket("ws://" + HOST + ":64002/signaling");
ws.binaryType = "arraybuffer";

const localConnectionPath1 = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] });
const localConnectionPath2 = new RTCPeerConnection({ iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] });

let dataChannel1 = localConnectionPath1.createDataChannel("xdpChannel1", {ordered: false, binaryType: 'arraybuffer', maxRetransmits: 0});
let dataChannel2 = localConnectionPath2.createDataChannel("xdpChannel2", {ordered: false, binaryType: 'arraybuffer', maxRetransmits: 0});
let dataChannel3 = null;
let dataChannel4 = null;
let dataChannel5 = null;
let dataChannel6 = null;

let messageCount = {
  "xdpChannel1": 0,
  "xdpChannel2": 0,
  "xdpChannel3": 0,
  "xdpChannel4": 0,
  "xdpChannel5": 0,
  "xdpChannel6": 0
};

let firstMessageTime = {
  "xdpChannel1": null,
  "xdpChannel2": null,
  "xdpChannel3": null,
  "xdpChannel4": null,
  "xdpChannel5": null,
  "xdpChannel6": null
};

let lastMessageTime = {
  "xdpChannel1": null,
  "xdpChannel2": null,
  "xdpChannel3": null,
  "xdpChannel4": null,
  "xdpChannel5": null,
  "xdpChannel6": null
};

let messageOrder = {
  "xdpChannel1": [],
  "xdpChannel2": [],
  "xdpChannel3": [],
  "xdpChannel4": [],
  "xdpChannel5": [],
  "xdpChannel6": []
};

function calculateOutOfOrderStats(orderArray) {
  let outOfOrderCount = 0;
  for (let i = 1; i < orderArray.length; i++) {
    if (orderArray[i] < orderArray[i - 1]) {
      outOfOrderCount++;
    }
  }
  return {
    outOfOrderCount: outOfOrderCount,
    outOfOrderRate: outOfOrderCount / orderArray.length
  };
}

function getStats() {
  let stats = {};
  for (let channel in messageCount) {
    stats[channel] = {
      messageCount: messageCount[channel],
      timeSpan: (lastMessageTime[channel] - firstMessageTime[channel]) / 1000,
      ...calculateOutOfOrderStats(messageOrder[channel])
    };

    console.log("Channel: ", channel);
    let notArrivedMessages = [];
    for (let i = 1; i <= 300; i++) {
      if (!messageOrder[channel].includes(i)) {
        notArrivedMessages.push(i);
      }
    }
    if (channel === "xdpChannel1") {
      console.log("Not arrived messages for xdpChannel1: ", notArrivedMessages);
    }
  }
  return stats;
}

function sendStatsToServer() {
  let stats = getStats();
  ws.send(JSON.stringify(stats));
}

function checkLastReceivedTime() {
  let currentTime = Date.now();
  let latestMessageTime = 0;
  for (let channel in lastMessageTime) {
    if (lastMessageTime[channel] && lastMessageTime[channel] > latestMessageTime) {
      latestMessageTime = lastMessageTime[channel];
    }
  }
  if (latestMessageTime != 0 && (currentTime - latestMessageTime > 2000)) {
    sendStatsToServer();

    // clear all stats
    for (let channel in messageCount) {
      messageCount[channel] = 0;
      firstMessageTime[channel] = null;
      lastMessageTime[channel] = null;
      messageOrder[channel] = [];
    }
  }
}

setInterval(checkLastReceivedTime, 2000);

localConnectionPath1.ondatachannel = async (event) => {
  if (event.channel.label === "xdpChannel1") {
    dataChannel1 = event.channel;

  } else if (event.channel.label === "xdpChannel3") {
    dataChannel3 = event.channel;
    dataChannel3.onopen = function(event) {
      // console.log("Data channel 3 is open on client.......");
      dataChannel3.onmessage = async (event) => {
        // console.log("Data channel 3 message: ", event.data);
        messageCount["xdpChannel3"]++;
        if (firstMessageTime["xdpChannel3"] === null) {
          firstMessageTime["xdpChannel3"] = Date.now();
        }
        lastMessageTime["xdpChannel3"] = Date.now();
        let id = new DataView(event.data).getUint32(0, true);
        messageOrder["xdpChannel3"].push(id);
      }
    }
  } else if (event.channel.label === "xdpChannel5") {
    dataChannel5 = event.channel;
    dataChannel5.onopen = function(event) {
      // console.log("Data channel 5 is open on client.......");
      dataChannel5.onmessage = async (event) => {
        // console.log("Data channel 5 message: ", event.data);
        messageCount["xdpChannel5"]++;
        if (firstMessageTime["xdpChannel5"] === null) {
          firstMessageTime["xdpChannel5"] = Date.now();
        }
        lastMessageTime["xdpChannel5"] = Date.now();
        let id = new DataView(event.data).getUint32(0, true);
        messageOrder["xdpChannel5"].push(id);
      }
    }
  }
  dataChannel1.onopen = function(event) {
    console.log("Data channel 1 is open on client.......");
    dataChannel1.send("Hello from client");
    dataChannel1.onmessage = (event) => {
      messageCount["xdpChannel1"]++;
      if (firstMessageTime["xdpChannel1"] === null) {
        firstMessageTime["xdpChannel1"] = Date.now();
      }
      lastMessageTime["xdpChannel1"] = Date.now();
    }
  };
}

localConnectionPath2.ondatachannel = async (event) => {
  if (event.channel.label === "xdpChannel2") {
    dataChannel2 = event.channel;
  } else if (event.channel.label === "xdpChannel4") {
    dataChannel4 = event.channel;
    dataChannel4.onopen = function(event) {
      // console.log("Data channel 4 is open on client.......");
      dataChannel4.onmessage = async (event) => {
        // console.log("Data channel 4 message: ", event.data);
        messageCount["xdpChannel4"]++;
        if (firstMessageTime["xdpChannel4"] === null) {
          firstMessageTime["xdpChannel4"] = Date.now();
        }
        lastMessageTime["xdpChannel4"] = Date.now();
        let id = new DataView(event.data).getUint32(0, true);
        messageOrder["xdpChannel4"].push(id);
      }
    }
  } else if (event.channel.label === "xdpChannel6") {
    dataChannel6 = event.channel;
    dataChannel6.onopen = function(event) {
      // console.log("Data channel 6 is open on client.......");
      dataChannel6.onmessage = async (event) => {
        // console.log("Data channel 6 message: ", event.data);
        messageCount["xdpChannel6"]++;
        if (firstMessageTime["xdpChannel6"] === null) {
          firstMessageTime["xdpChannel6"] = Date.now();
        }
        lastMessageTime["xdpChannel6"] = Date.now();
        let id = new DataView(event.data).getUint32(0, true);
        messageOrder["xdpChannel6"].push(id);
      }
    }
  }
  dataChannel2.onopen = function(event) {
    dataChannel2.send("Hello from client");
    // console.log("Data channel 2 is open on client.......");
    dataChannel2.onmessage = async (event) => {
      messageCount["xdpChannel2"]++;
      if (firstMessageTime["xdpChannel2"] === null) {
        firstMessageTime["xdpChannel2"] = Date.now();
      }
      lastMessageTime["xdpChannel2"] = Date.now();
      let id = new DataView(event.data).getUint32(0, true);
      messageOrder["xdpChannel2"].push(id);
    }

  }
}


ws.onmessage = async (event) => {
  let sd = event.data.slice(2);
  if (event.data[0] === '1') {
    try {
      localConnectionPath1.setRemoteDescription(JSON.parse(atob(sd)));
    } catch (e) {
      alert(e);
    }
  } else if (event.data[0] === '2') {
    try {
      localConnectionPath2.setRemoteDescription(JSON.parse(atob(sd)));
    } catch (e) {
      alert(e);
    }
  } else if (event.data[0] === 'a') {
    const cand = JSON.parse(event.data.slice(2)).candidate;
    const candidateInfo = {
      candidate: cand,
      sdpMLineIndex: 0,
      sdpMid: ""
    };
    localConnectionPath1.addIceCandidate(new RTCIceCandidate(candidateInfo))
  } else if (event.data[0] === 'b') {
    const cand = JSON.parse(event.data.slice(2)).candidate;
    const candidateInfo = {
      candidate: cand,
      sdpMLineIndex: 0,
      sdpMid: ""
    };
    localConnectionPath2.addIceCandidate(new RTCIceCandidate(candidateInfo))
  } else if (event.data[0] === 'w') {
    server_screen_width = parseInt(event.data.slice(2));
    width_is_set = true;
  } else if (event.data[0] === 'h') {
    server_screen_height = parseInt(event.data.slice(2));
    height_is_set = true;
  } 
  else if (event.data instanceof ArrayBuffer) {
    let arr = new Uint8Array(event.data);
    if (arr[0] === 0 && arr[1] === 0 && arr[2] === 0 && arr[3] === 1) {
      
      g_sps_pps = arr;
    }
  }

}

ws.onopen = () => {
  localConnectionPath1.createOffer().then(offer => {
    localConnectionPath1.setLocalDescription(offer);
    ws.send("s1:" + btoa(JSON.stringify(offer)));
  });
  localConnectionPath2.createOffer().then(offer => {
    localConnectionPath2.setLocalDescription(offer);
    ws.send("s2:" + btoa(JSON.stringify(offer)));
  });
}

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}