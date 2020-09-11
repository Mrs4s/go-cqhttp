/*!
 * jQuery infiniteScroll v1.0.0
 * 无限滚动插件
 * http://www.thinkcmf.com
 * MIT License
 * by Dean(老猫)
 */
;(function($){
	$.fn.infiniteScroll=function(options){
		var opts = $.extend({},$.fn.infiniteScroll.defaults, options); 
		var url = location.href;
		var $loading = $(opts.loading);
		$loading.hide();
		return this.each(function(){
			var $document=$(document);
			var $window =$(window);
			var $this=$(this);
			var page=opts.page;
			$(window).scroll(function(){
				if($this.data('loading')) return;
				if($document.scrollTop() > $this.position().top-$window.height()){
					$this.data('loading',true);
					$loading.show();
					var data={};
					page++;
					if(page>opts.total_pages){
						$loading.hide();
						opts.finish();
						return;
					}
					data[opts.pageParam]=page;
					$.ajax({
						url:url,
						data:data,
						type:'GET',
						dateType:'html',
						success:function(content){
							opts.success(content,page);
						},
						error:function(){
							opts.error();
						},
						complete:function(){
							$loading.hide();
							$this.data('loading',false);
						}
					});
				}
			});
			
		});
	};
	$.fn.infiniteScroll.defaults = {
		pageParam:'p',
		loading:'.js-infinite-scroll-loading',
		page:1,
		success:function(){},
		finish:function(){},
		error:function(){},
	}; 
})(jQuery); 