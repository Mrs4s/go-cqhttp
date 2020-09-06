//
//  这是一份测试脚本
//

function on_create() {
    var a = message.NewText("Nigero!");
    var m = message.NewSendingMessage();
    m.Elements.push(a);
    CQClient.SendPrivateMessage(1078007266, m);
}
