// スレッド閲覧画面
// コメント投稿ボタン
$('.btn-post-comment').on('click', () => {
    var commentModal = new bootstrap.Modal(document.getElementById('commentModal'));
    let comment = $('.comment-textarea').val();
    if (comment) {
        var comments = comment.split(/\n/);
        commentModal.show();
        if (comment.length > 500) {
            $('#commentModal p').text('500文字以内で投稿してください。');
            $('#commentModal .btn-comment-submit').hide();
        } else {
            comments.forEach(c => {
                $('#commentModal p').append(escape_html(c));
                $('#commentModal p').append('<br>');
            });
        }
    }
});
// 投稿モーダルフォーム初期化
$('#commentModal').on('hidden.bs.modal', function (e) {
    $('#commentModal p').empty();
    $('#commentModal .message').html('');
    $('#commentModal .btn-comment-submit').show();
})
$('.btn-comment-submit').on('click', (e) => {
    // 返信の時は >>commentId と記載される想定
    let comment = $('.comment-textarea').val().trim();
    if (comment === '') return;

    // 文字数制限500文字まで
    if (comment.length > 500) {
        $('#commentModal .message').text('500文字以内で投稿してください。');
        $('#commentModal p').text('');
        return;
    }
    $(e.target).prop('disabled', true);
    let form = $('form[name="comment"]');
    url = $(e.target).data('postCommentUrl');
    $.ajax({
        type: 'POST',
        url: url,
        dataType: 'json',
        data: form.serialize(),
    }).
    then(
        (data, textStatus, jqXHR) => {
            if (data.success) {
                location.reload()
            } else {
                $('#commentModal .message').text('エラーが発生しました。');
                $(e.target).prop('disabled', false);
            }
        },
        (xhr, status, err) => {
            let message = err;
            $(e.target).prop('disabled', false);
            if (xhr.status == 401) {
                message = 'ログインしなおしてください。';
            }
            if (xhr.status == 403) {
                message = '権限がありません。';
            }
            $('#commentModal .message').html('エラーが発生しました。<br/>' + message);
        },
    );
});

// いいね機能
function postLike(good, commentId, target, csrfToken, url) {
    $.ajax({
        type: 'POST',
        url: url,
        dataType: 'json',
        data: {
            comment_id: commentId,
            good: good,
            bad: !good,
            _csrf: csrfToken,
        },
    }).
    then(
        (data, textStatus, jqXHR) => {
            if (data.success) {
                $('.good-count-' + commentId).text(data.goodCount);
                $('.bad-count-' + commentId).text(data.badCount);
                // アイコンの変更
                // fa-thumbs-up, fa-thumbs-down
                let oppositeClass = '';
                if (good) {
                    oppositeClass = '.fa-thumbs-down';
                } else {
                    oppositeClass = '.fa-thumbs-up';
                }
                if (target.hasClass('fa-solid')) {
                    target.removeClass('fa-solid');
                    target.addClass('fa-regular');

                    if (data.isChangedLike) {
                        $('.progressbar-' + commentId + ' ' + oppositeClass).removeClass('fa-regular');
                        $('.progressbar-' + commentId + ' ' + oppositeClass).addClass('fa-solid');
                    }
                } else {
                    target.addClass('fa-solid');
                    target.removeClass('fa-regular')
                    if (data.isChangedLike) {
                        $('.progressbar-' + commentId + ' ' + oppositeClass).removeClass('fa-solid');
                        $('.progressbar-' + commentId + ' ' + oppositeClass).addClass('fa-regular');
                    }
                }

                let total = data.goodCount + data.badCount;
                let percent = 0;
                if (total != 0) {
                    percent = Math.ceil(data.goodCount / total  * 100);
                }

                $('.progressbar-' + commentId + ' .progress-bar').css('width', percent + '%');
                return;
            } else {
                alert('エラーが発生しました。')
            }
        },
        (xhr, status, err) => {
            let message = err;
            if (xhr.status == 401) {
                message = 'ログインしなおしてください。';
            }
            if (xhr.status == 403) {
                message = '権限がありません。';
            }
            alert('エラーが発生しました。');
        },
    );
}
// 返信アイコン押下
$('.btn-reply').on('click', (e) => {
    let parentId = $(e.target).data('parentId');
    let parentNum = $(e.target).data('parentNum');

    $('.modal-body .reply-message').text('>> ' + parentNum);
    $('input[name="parent_id"]').val(parentId);
    $('input[name="parent_number"]').val(parentNum);
});
// 返信モーダルフォーム初期化
$('#replyModal').on('hide.bs.modal', function (e) {
    $('#replyModal textarea').val('');
    $('input[name="parent_id"]').val('');
    $('input[name="parent_number"]').val('');
})

$('.btn-reply-submit').on('click', (e) => {
    let replyForm = $('form[name="reply"]');
    let reply = $('.reply-textarea').val().trim();
    if (reply === '') return;

    // 文字数制限500文字まで
    if (reply.length > 500) {
        $('.reply-message').html('500文字以内で返信してください。');
        return;
    }

    url = $(e.target).data('postReplyUrl');
    $(e.target).prop('disabled', true);
    $.ajax({
        type: 'POST',
        url: url,
        dataType: 'json',
        data: replyForm.serialize(),
    }).
    then(
        (data, textStatus, jqXHR) => {
            if (data.success) {
                location.reload()
            } else {
                $('.reply-message').text('エラーが発生しました。');
                $(e.target).prop('disabled', false);
            }
        },
        (xhr, status, err) => {
            let message = err;
            $(e.target).prop('disabled', false);
            if (xhr.status == 401) {
                message = 'ログインしなおしてください。';
            }
            if (xhr.status == 403) {
                message = '権限がありません。';
            }
            $('.reply-message').html('エラーが発生しました。<br/>' + message);
        },
    );
});

$('.btn-image-submit').on('click', () => {
    if ($('#imageModalFile').val() == '') return;

    let file = $('#imageModalFile').prop('files')[0];
    if (!/\.(jpg|jpeg|png|gif|JPG|JPEG|PNG|GIF)$/.test(file.name) || !/(jpg|jpeg|png|gif)$/.test(file.type)) {
        alert('JPG、GIF、PNGファイルの画像を添付してください。');
        //添付された画像ファイルが１M以下か検証する
    } else if (5*1024*1024 < file.size) {
        alert('5MB以下の画像を添付してください。');
    } else {
        $('form[name="image"]').submit();
    }
});

$('.btn-giin-submit').on('click', (e) => {
    let form = $('form[name="giin"]');
    let comment = $('.giin-textarea').val().trim();
    if (comment === '') return;

    // 文字数制限500文字まで
    if (comment.length > 500) {
        $('.giin-message').html('500文字以内で返信してください。');
        return;
    }

    url = $(e.target).data('postGiinUrl');
    $(e.target).prop('disabled', true);
    $.ajax({
        type: 'POST',
        url: url,
        dataType: 'json',
        data: form.serialize(),
    }).
    then(
        (data, textStatus, jqXHR) => {
            if (data.success) {
                location.reload()
            } else {
                $('.giin-message').text('エラーが発生しました。');
                $(e.target).prop('disabled', false);
            }
        },
        (xhr, status, err) => {
            let message = err;
            $(e.target).prop('disabled', false);
            if (xhr.status == 401) {
                message = 'ログインしなおしてください。';
            }
            if (xhr.status == 403) {
                message = '権限がありません。';
            }
            $('.giin-message').html('エラーが発生しました。<br/>' + message);
        },
    );
});

$('.btn-giin-del-submit').on('click', (e) => {
    let form = $('form[name="giin-del"]');

    url = $(e.target).data('postGiinDelUrl');
    $(e.target).prop('disabled', true);
    $.ajax({
        type: 'POST',
        url: url,
        dataType: 'json',
        data: form.serialize(),
    }).
    then(
        (data, textStatus, jqXHR) => {
            if (data.success) {
                location.reload()
            } else {
                $('.giin-del-message').text('エラーが発生しました。');
                $(e.target).prop('disabled', false);
            }
        },
        (xhr, status, err) => {
            let message = err;
            $(e.target).prop('disabled', false);
            if (xhr.status == 401) {
                message = 'ログインしなおしてください。';
            }
            if (xhr.status == 403) {
                message = '権限がありません。';
            }
            $('.giin-del-message').html('エラーが発生しました。<br/>' + message);
        },
    );
});

function escape_html (string) {
    if(typeof string !== 'string') {
        return string;
    }
    return string.replace(/[&'`"<>]/g, function(match) {
        return {
            '&': '&amp;',
            "'": '&#x27;',
            '`': '&#x60;',
            '"': '&quot;',
            '<': '&lt;',
            '>': '&gt;',
        }[match]
    });
}