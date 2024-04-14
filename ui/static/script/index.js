const imgFormat = 'png';
const defaultAvatar = 'c6312289-d30d-4456-acdf-6efd18677e84';

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
    //document.body.addEventListener('click', nobox);
    scrollBut.addEventListener('click', scrollUp);
    boxElem.addEventListener('click', nobox);

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
    if (size == 'f') {
        size = '';
    } else {
        size = '_' + size;
    }

    if (!userAvatar) userAvatar = defaultAvatar;

    return `/uploads/${userAvatar}${size}.png`;
}

function formatDate(date) {
    if (!date) return '';

    date = new Date(date);
    let month = new String(date.getMonth() + 1);
    if (month.length < 2) {
        month = '0' + month;
    }

    let day = new String(date.getDate());
    if (day.length < 2) {
        day = '0' + day;
    }
    return `${date.getFullYear()}-${month}-${day}`;
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