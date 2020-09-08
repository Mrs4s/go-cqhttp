//
//  这是一份测试脚本
//

function on_create() {
}

function on_missed_action(action, params) {
}

function on_missed_cqcode(action, params) {
}

CQClient.OnPrivateMessage(function (client, privateMessage) {
    var a = message.NewText("test");
    var m = message.NewSendingMessage();
    m.Elements.push(a);
    client.SendPrivateMessage(privateMessage.Sender.Uin, m);
});