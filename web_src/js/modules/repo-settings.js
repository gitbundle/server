import { initJqDropdownClose } from '../common/dropdown.js'
import RepoSettingsCollabTeam from '../components/RepoSettingsCollabTeam.svelte'
import $ from '../features/jquery.js'
import { createMonaco } from './codeeditor.js'

const { appSubUrl, csrfToken } = window.config

export function initRepoSettingsCollaboration(selector) {
  if (!$(selector).length) return

  const dropdown = '[jq-access-mode]'
  initJqDropdownClose(dropdown, undefined)

  // Change collaborator access mode
  $(`${selector} ${dropdown} [jq-item]`).on('click', function () {
    const $dropdown = $(this).closest(dropdown)
    const $text = $dropdown.find('label > [jq-text]')

    const value = $(this).data('value')
    const lastValue = $dropdown.data('last-value')

    $.post($dropdown.data('url'), {
      _csrf: csrfToken,
      uid: $dropdown.data('uid'),
      mode: value
    }).fail(() => {
      $text.text('(error)') // prevent from misleading users when error occurs
      $dropdown.attr('data-last-value', lastValue)
    })
    $text.text($(this).data('text'))
    $dropdown.attr('data-last-value', value)
    $dropdown.removeClass('dropdown-open')
  })

  // $(`${selector} ${dropdown}`).each((_, e) => {
  //   const $dropdown = $(e)
  //   const $text = $dropdown.find('> .text')
  //   $dropdown.dropdown({
  //     action(_text, value) {
  //       const lastValue = $dropdown.attr('data-last-value')
  //       $.post($dropdown.attr('data-url'), {
  //         _csrf: csrfToken,
  //         uid: $dropdown.attr('data-uid'),
  //         mode: value
  //       }).fail(() => {
  //         $text.text('(error)') // prevent from misleading users when error occurs
  //         $dropdown.attr('data-last-value', lastValue)
  //       })
  //       $dropdown.attr('data-last-value', value)
  //       $dropdown.dropdown('hide')
  //     },
  //     onChange(_value, text, _$choice) {
  //       $text.text(text) // update the text when using keyboard navigating
  //     },
  //     onHide() {
  //       // set to the really selected value, defer to next tick to make sure `action` has finished its work because the calling order might be onHide -> action
  //       setTimeout(() => {
  //         const $item = $dropdown.dropdown(
  //           'get item',
  //           $dropdown.attr('data-last-value')
  //         )
  //         if ($item) {
  //           $dropdown.dropdown(
  //             'set selected',
  //             $dropdown.attr('data-last-value')
  //           )
  //         } else {
  //           $text.text('(N/A)') // prevent from misleading users when the access mode is undefined
  //         }
  //       }, 0)
  //     }
  //   })
  // })
}

export function initRepoSearchBox(selector, data) {
  const $els = $(selector)
  if (!$els.length) return
  const el = $els[0]

  const pageData = window.config.pageData[data]
  new RepoSettingsCollabTeam({
    target: el,
    props: { pageData }
  })
}

// export function initRepoSettingSearchTeamBox() {
//   const $searchTeamBox = $('#search-team-box')
//   $searchTeamBox.search({
//     minCharacters: 2,
//     apiSettings: {
//       url: `${appSubUrl}/org/${$searchTeamBox.data(
//         'org'
//       )}/teams/-/search?q={query}`,
//       headers: { 'X-Csrf-Token': csrfToken },
//       onResponse(response) {
//         const items = []
//         $.each(response.data, (_i, item) => {
//           const title = `${item.name} (${item.permission} access)`
//           items.push({
//             title
//           })
//         })

//         return { results: items }
//       }
//     },
//     searchFields: ['name', 'description'],
//     showNoResults: false
//   })
// }

export function initRepoSettingGitHook(selector) {
  if ($(selector).length === 0) return
  const filename = document.querySelector('[jq-hook-filename]').textContent
  const _promise = createMonaco($('#content')[0], filename, {
    language: 'shell'
  })
}

export function initRepoSettingBranches(selector) {
  // Branches
  if ($(selector).length > 0) {
    // initRepoCommonFilterSearchDropdown('.protected-branches .dropdown')
    $(
      '[jq-enable-protection], [jq-enable-whitelist], [jq-enable-statuscheck]'
    ).on('change', function () {
      if (this.checked) {
        $($(this).data('target')).removeClass('disabled')
      } else {
        $($(this).data('target')).addClass('disabled')
      }
    })
    $('[jq-disable-whitelist]').on('change', function () {
      if (this.checked) {
        $($(this).data('target')).addClass('disabled')
      }
    })
  }
}
