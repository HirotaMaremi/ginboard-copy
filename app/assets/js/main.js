// 画面TOPに戻る
!function (win, doc, $) {
    var pageTopPos = 200;
    if ($('.top-back-btn').length > 0) {
        $pageTopBtn = $('.top-back-btn');
        $(win).on('load scroll', () => {
            if ($(this).scrollTop() > pageTopPos) {
                $pageTopBtn.fadeIn();
            } else {
                $pageTopBtn.fadeOut();
            }
        });
        $pageTopBtn.on('click', () => {
           $('bady,html').animate({
               scrollTop: 0
           }, 300);
           return false;
        });
    }
} (window, document, jQuery);

// 最下部に移動
function pageBottom(){
    let elm = document.documentElement;
    // scrollHeight ページの高さ clientHeight ブラウザの高さ
    let bottom = elm.scrollHeight - elm.clientHeight;
    // 垂直方向へ移動
    window.scroll({
        top: bottom,
        behavior: 'smooth'
    });

}

!function (win, doc, $) {
    let elm = document.documentElement;
    // let bottom = elm.scrollHeight - elm.clientHeight;
    if ($('.js-scroll-bottom').length > 0) {
        $bottomScrollBtn = $('.js-scroll-bottom');
        $(win).on('load scroll', () => {
            if ($(this).scrollTop() + 20 < elm.scrollHeight - elm.clientHeight) {
                $bottomScrollBtn.fadeIn();
            } else {
                $bottomScrollBtn.fadeOut();
            }
        });
        $bottomScrollBtn.on('click', () => {
            $('bady,html').animate({
                scrollTop: elm.scrollHeight - elm.clientHeight
            }, 300);
            return false;
        });
    }
} (window, document, jQuery);