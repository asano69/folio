// 画像表示に関する機能
// 現時点では基本的な表示のみだが、将来的にズームや回転などを追加予定

export function initImageDisplay() {
  const image = document.getElementById('page-image') as HTMLImageElement;

  if (!image) {
    return;
  }

  // 画像読み込み完了時の処理
  image.addEventListener('load', () => {
    console.log('Image loaded successfully');
  });

  // 画像読み込みエラー時の処理
  image.addEventListener('error', () => {
    console.error('Failed to load image');
  });
}
