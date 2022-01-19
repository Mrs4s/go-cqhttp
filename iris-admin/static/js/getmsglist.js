$(document).ready(function () {
    var url = "/admin/qq/getmsglistforajax?uin=" + getUrlParam("uin")
    $("#msgsend").click(function () {
        var username = $("#msgtext").data("username");
        var text = $("#msgtext").val();
        var tmp = `<div class="row" style="text-align:right;margin-left:15px;margin-right:15px;">
<div class="row">
<i class="fa fa-arrow-left"></i>{0}
</div>
<p>{1}</p>
</div>`;
        var htmlobj = String.format(tmp, username, text);
        var msgtype = $("#msgtext").data("type");
        $("#msgbox .box-body").append(htmlobj);
        $.post("/admin/qq/sendmsg",
            {
                type: msgtype,
                uin: getUrlParam("uin"),
                text: text,
            }
        );
        $("#msgtext").val("");
    });
    t = setInterval(getmsg, 1000);

    function getmsg() {
        if (getUrlParam("uin") == null) {
            clearInterval(t)
            return
        }
        $.get(url, function (data, status) {
            if (data.code === 200) {
                htmlobj = data.html;
                $("#msgbox .box-body").html(htmlobj);
            }
        })
    }


    //获取url中的参数
    function getUrlParam(name) {
        var reg = new RegExp("(^|&)" + name + "=([^&]*)(&|$)"); //构造一个含有目标参数的正则表达式对象
        var r = window.location.search.substr(1).match(reg);  //匹配目标参数
        if (r != null) return unescape(r[2]);
        return null; //返回参数值
    }

    String.format = function () {
        if (arguments.length == 0)
            return null;
        var str = arguments[0];
        for (var i = 1; i < arguments.length; i++) {
            var re = new RegExp('\\{' + (i - 1) + '\\}', 'gm');
            str = str.replace(re, arguments[i]);
        }
        return str;
        /*
        调用方式：
            var info = "我喜欢吃{0}，也喜欢吃{1}，但是最喜欢的还是{0},偶尔再买点{2}。";
            var msg=String.format(info , "苹果","香蕉","香梨")
            alert(msg);
            输出:我喜欢吃苹果，也喜欢吃香蕉，但是最喜欢的还是苹果,偶尔再买点香梨。
        */
    };
});