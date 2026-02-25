const imgFormat = `png`;
const defaultAvatarLink = `/static/img/avatar`;
const photoThumbnail = `${window.location.href}#`;

let scrollBut, boxElem;

const user = {
    id: 0,
    name: '',
    lastname: '',
    avatar: 0,
    description: '',
    newsCount: 0
};

let currentUser = {
    id: 0,
    name: '',
    lastname: '',
    avatar: 0,
    description: '',
    newsCount: 0,
    newsCountLoaded: 0,
    newsPartLoaded: 0,
    newsLastId: 0
};

window.addEventListener('load', init);

async function init() {
    scrollBut = $('#scroll-but');
    boxElem = $('#box');
    window.addEventListener('scroll', onScroll);
    scrollBut.addEventListener('click', scrollUp);
    boxElem.addEventListener('click', nobox);
    document.body.addEventListener('keydown', keyDownHandler);

    const cookieId = getCookie('id');
    if (cookieId) {
        try {
            const response = await fetch('/api/ping');

            if (response.status == 200) {
                const resJSON = await response.json();
                user.id = resJSON.id;
            } else {

            }
        } catch (error) {

        }

        if (user.id) {
            await getUserData(user.id);
        }
    }

    initRouter();
}

async function getUserData(id) {
    if (!id) return {};

    const req = await fetch(`/api/users/${id}`);
    const json = await req.json();

    user.name = json.name;
    user.lastname = json.lastname;
    if (json.avatar) user.avatar = json.avatar;
    if (json.newsCount) user.newsCount = json.newsCount;
    if (json.description) user.description = json.description;
}

// Utils

function getUserAvatarLink(userAvatar, size = 's') {
    size = '_' + size;

    if (!userAvatar) {
        return `${defaultAvatarLink}${size}.png`;
    }

    return `/uploads/${userAvatar}${size}.png`;
}

function formatDate(date, withTime=false) {
    if (!date) return '';
    date = new Date(date);

    let result = `${formatPart(date.getFullYear())}-${formatPart(date.getMonth() + 1)}-${formatPart(date.getDate())}`;

    if (withTime) {
        result += ` ${formatPart(date.getHours())}:${formatPart(date.getMinutes())}`;
    }

    return result;

    function formatPart(s) {
        s = new String(s);
        if (s.length < 2) {
            s = '0' + s;
        }
        return s;
    }
}

function getCookie(name) {
    const matches = document.cookie.match(new RegExp(
        '(?:^|; )' + name.replace(/([\.$?*|{}\(\)\[\]\\\/\+^])/g, '\\$1') + '=([^;]*)'
    ));

    return matches ? decodeURIComponent(matches[1]) : undefined;
}

function $(s) {
    return document.querySelector(s);
}

// UI

function showDialog(name) {
    $('#dialogs-cont').style.display = 'none';

    let dialogs = $('#dialogs-cont').children;

    for (let i of dialogs) {
        i.style.display = 'none';
    }

    if (!name) { return; }

    $('#dialogs-cont').style.display = 'block';
    $('#dialog-' + name).style.display = 'block';
}

function onScroll() {
    const elem = document.documentElement;

    if ((elem.scrollTop > 0) && ((elem.scrollTop > elem.clientHeight * 1.5) || (elem.scrollHeight - elem.scrollTop === elem.clientHeight))) {
        scrollBut.style.display = 'block';
    } else {
        scrollBut.style.display = 'none';
    }
}

function scrollUp() {
    document.documentElement.scrollTo({ top: 0, left: 0, behavior: 'smooth' });
}

function lightbox(a) {
    document.body.style.overflow = 'hidden';
    boxElem.style.display = 'block';
    bimg.src = a;
}

function nobox() {
    document.body.style.overflow = '';
    boxElem.style.display = 'none';
}

function keyDownHandler(e) {
	if (e.key === 'Escape') {
        nobox();
    }
}