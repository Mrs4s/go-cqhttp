//
//  这是一份测试脚本
//

function on_create() {
}

function on_missed_action(action, params) {
    if(action === "test_action")
    {
        var a = message.NewText(params.msg);
        var m = message.NewSendingMessage();
        m.Elements.push(a);
        CQClient.SendPrivateMessage(0, m); // 0 为目标qq号
        return JSON.stringify({status:"ok"});
    }
}

function on_missed_cqcode(action, params) {
}