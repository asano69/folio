// UI コンポーネントとイベント処理
// 将来的にモーダルやトーストなどの UI 要素を追加予定

export function showToast(message: string, type: 'info' | 'success' | 'error' = 'info') {
  // 現時点ではコンソールログのみ
  console.log(`[${type.toUpperCase()}] ${message}`);

  // 将来的にはトースト通知を実装
}

export function createModal(content: string, title?: string) {
  // 将来的にモーダルダイアログを実装
  console.log('Modal:', title, content);
}
