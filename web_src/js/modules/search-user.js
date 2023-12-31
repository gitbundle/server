import $ from 'jquery'
import SearchUser from '../components/SearchUser.svelte'
// import {htmlEscape} from 'escape-goat';

export function initSearchUserBox(selector, pageDataKey) {
  const $els = $(selector)
  if (!$els.length) return
  const el = $els[0]

  const pageData = window.config.pageData[pageDataKey]
  new SearchUser({
    target: el,
    props: { pageData }
  })
}

// const {appSubUrl} = window.config;

// export function initCompSearchUserBox() {
//   const $searchUserBox = $('#search-user-box');
//   $searchUserBox.search({
//     minCharacters: 2,
//     apiSettings: {
//       url: `${appSubUrl}/user/search?q={query}`,
//       onResponse(response) {
//         const items = [];
//         const searchQueryUppercase = $searchUserBox.find('input').val().toUpperCase();
//         $.each(response.data, (_i, item) => {
//           let title = item.login;
//           if (item.full_name && item.full_name.length > 0) {
//             title += ` (${htmlEscape(item.full_name)})`;
//           }
//           const resultItem = {
//             title,
//             image: item.avatar_url
//           };
//           if (searchQueryUppercase === item.login.toUpperCase()) {
//             items.unshift(resultItem);
//           } else {
//             items.push(resultItem);
//           }
//         });

//         return {results: items};
//       }
//     },
//     searchFields: ['login', 'full_name'],
//     showNoResults: false
//   });
// }
