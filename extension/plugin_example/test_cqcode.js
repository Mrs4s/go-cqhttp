//
//  这是一份测试脚本
//

function on_create() {
}


function on_missed_cqcode(code, params) {
    if(code === "test_code")
    {
        return message.NewText(params.content);
    }
}

function on_missed_action() {
}