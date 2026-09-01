import { useEffect, useState } from 'react'
import { IconArrowLeft, IconArrowRight, IconX } from './icons.tsx'
import { Photo } from './media.tsx'

// 1 large + 4 thumb grid (§6.3). Renders neutral placeholders until real
// photos exist; the lightbox activates only when real photos are present.
export default function Gallery({ images = [], alt = '' }: { images?: string[]; alt?: string }) {
  const [active, setActive] = useState(0)
  const [box, setBox] = useState(false)
  const hasReal = images.length > 0

  useEffect(() => {
    if (!box) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setBox(false)
      if (e.key === 'ArrowRight') setActive((i) => (i + 1) % images.length)
      if (e.key === 'ArrowLeft') setActive((i) => (i - 1 + images.length) % images.length)
    }
    document.addEventListener('keydown', onKey)
    document.body.style.overflow = 'hidden'
    return () => { document.removeEventListener('keydown', onKey); document.body.style.overflow = '' }
  }, [box, images.length])

  /*
   * Always reserve 4 thumbnail slots; wrap around images if fewer than 4 are
   * provided so no slot is left as an unpopulated placeholder.
   *
   * The wrap has to be carried into the CLICK as well, not just the render.
   * The thumb rendered at slot 3 of a 3-image set is images[0], but the button
   * used to call setActive(3) — and `images[3]` is undefined, so clicking the
   * last thumbnail replaced the main photo with a grey placeholder and the
   * lightbox opened on nothing. Storing the wrapped index instead means the
   * main image follows the thumbnail that was actually clicked.
   *
   * Latent until 1 Sep 2026: every gallery on the site had at least four
   * photos, so no slot ever wrapped. The first venue to ship with three was
   * Amaltas's banquet hall, and the 3-image fallback in `banquetImages()`
   * means any venue without its own folder would have hit it too.
   */
  const slotFor = (i: number): number => (images.length ? i % images.length : i)
  const thumbs = Array.from({ length: 4 }, (_, i) => (images.length ? images[slotFor(i)] : undefined))

  return (
    <>
      <div className="gallery">
        {hasReal ? (
          <button className="gallery__main" onClick={() => setBox(true)} aria-label="Open gallery">
            <Photo src={images[active]} ratio="16 / 11" label={alt} alt={`${alt} — photo ${active + 1}`} />
          </button>
        ) : (
          <div className="gallery__main gallery__main--static">
            <Photo ratio="16 / 11" label={alt} alt={alt} />
          </div>
        )}

        <div className="gallery__thumbs">
          {thumbs.map((src, i) => (
            hasReal ? (
              <button
                key={i}
                className={`gallery__thumb ${slotFor(i) === active ? 'is-active' : ''}`}
                onClick={() => setActive(slotFor(i))}
                aria-label={`View photo ${i + 1}`}
              >
                <Photo src={src} fill label={alt} alt={`${alt} thumbnail ${i + 1}`} />
              </button>
            ) : (
              <div className="gallery__thumb" key={i}>
                <Photo fill label={alt} alt={alt} />
              </div>
            )
          ))}
        </div>
      </div>

      {box && hasReal && (
        <div className="lightbox" role="dialog" aria-modal="true" aria-label={`${alt} gallery`} onClick={() => setBox(false)}>
          <button className="lightbox__close" onClick={() => setBox(false)} aria-label="Close gallery"><IconX /></button>
          <button className="lightbox__nav lightbox__nav--prev" onClick={(e) => { e.stopPropagation(); setActive((i) => (i - 1 + images.length) % images.length) }} aria-label="Previous"><IconArrowLeft /></button>
          <img className="lightbox__img" src={images[active]} alt={`${alt} — photo ${active + 1}`} onClick={(e) => e.stopPropagation()} />
          <button className="lightbox__nav lightbox__nav--next" onClick={(e) => { e.stopPropagation(); setActive((i) => (i + 1) % images.length) }} aria-label="Next"><IconArrowRight /></button>
          <span className="lightbox__count">{active + 1} / {images.length}</span>
        </div>
      )}
    </>
  )
}
