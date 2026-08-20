import { useState, useEffect } from 'react'

interface PhotoProps {
  src?: string | undefined
  label?: string
  ratio?: string // e.g. '4 / 3'
  alt?: string
  className?: string
  /**
   * Fill the parent's height instead of reserving a fixed ratio. Used where the
   * parent is a grid cell whose height is set by a sibling — a fixed ratio there
   * leaves the cell part-empty. Sets a class rather than an inline style so CSS
   * (including media queries) can still take over.
   */
  fill?: boolean
}

/**
 * Photo — an image slot with a reserved aspect-ratio.
 * Renders a neutral --bg-warm placeholder (label centered) until a real photo
 * URL is supplied, so real photography drops in with zero layout shift (§5).
 */
export function Photo({ src, label = 'Quadis', ratio = '4 / 3', alt, className = '', fill = false }: PhotoProps) {
  const [failed, setFailed] = useState(false)
  const show = src && !failed
  return (
    <div
      className={`photo ${fill ? 'photo--fill' : ''} ${className}`}
      style={fill ? undefined : { aspectRatio: ratio }}
    >
      {show ? (
        <img className="photo__img" src={src} alt={alt || label} loading="lazy" onError={() => setFailed(true)} />
      ) : (
        <div className="photo__ph" role="img" aria-label={alt || label}>
          <span className="photo__ph-label">{label}</span>
        </div>
      )}
    </div>
  )
}

interface HeroMediaProps {
  src?: string | undefined
  alt?: string
}

/**
 * HeroMedia — full-bleed background for a hero/scrim band.
 * With a real photo: <img> under the parent's .scrim overlay.
 * Without: a dark gradient placeholder so light hero text keeps ≥4.5:1 contrast
 * (§7). Same container either way → zero layout change when a photo arrives.
 */
export function HeroMedia({ src, alt = '' }: HeroMediaProps) {
  const [failed, setFailed] = useState(false)
  if (src && !failed) {
    return <img className="hero-media" src={src} alt={alt} aria-hidden={alt ? undefined : true} onError={() => setFailed(true)} />
  }
  return <div className="hero-media hero-media--ph" aria-hidden="true" />
}

interface HeroShowcaseProps {
  images?: string[]
  intervalMs?: number
}

/**
 * HeroShowcase — dynamic full-bleed hero carousel with smooth 3s crossfades
 * and smart sequential rotation on refresh.
 */
export function HeroShowcase({ images = [], intervalMs = 3000 }: HeroShowcaseProps) {
  const [index, setIndex] = useState(() => {
    if (typeof window === 'undefined' || !images.length) return 0
    const saved = Number(localStorage.getItem('quadis_hero_last_idx') || 0)
    const next = (saved + 1) % images.length
    localStorage.setItem('quadis_hero_last_idx', String(next))
    return next
  })
  const [prevIndex, setPrevIndex] = useState(index)

  useEffect(() => {
    if (!images.length || images.length <= 1) return
    const timer = setInterval(() => {
      setIndex((current) => {
        setPrevIndex(current)
        return (current + 1) % images.length
      })
    }, intervalMs)
    return () => clearInterval(timer)
  }, [images.length, intervalMs])

  const handleDotClick = (i: number) => {
    if (i === index) return
    setPrevIndex(index)
    setIndex(i)
  }

  if (!images.length) {
    return <div className="hero-media hero-media--ph" aria-hidden="true" />
  }

  return (
    <>
      {images.map((img, i) => {
        const isActive = i === index
        const isPrev = i === prevIndex && !isActive
        return (
          <img
            key={img}
            className="hero-media"
            src={img}
            alt={`Quadis property showcase ${i + 1}`}
            style={{
              opacity: isActive || isPrev ? 1 : 0,
              transition: isActive ? 'opacity 1s ease-in-out' : 'none',
              zIndex: isActive ? 0 : isPrev ? -1 : -2,
            }}
          />
        )
      })}
      {images.length > 1 && (
        <div className="hero-dots">
          {images.map((_, i) => (
            <button
              key={i}
              type="button"
              aria-label={`Show photo ${i + 1}`}
              className={`hero-dot ${i === index ? 'is-active' : ''}`}
              onClick={() => handleDotClick(i)}
            />
          ))}
        </div>
      )}
    </>
  )
}

interface HeroVideoShowcaseProps {
  videoUrl?: string
  posterUrl?: string
}

/* The conditions under which this device should never be sent the hero video.
   Kept as a list so the effect below can subscribe to exactly what it tests. */
const HERO_VIDEO_OFF_QUERIES = ['(prefers-reduced-motion: reduce)', '(max-width: 768px)']

/**
 * Should the hero video be withheld from this device?
 *
 * `/videos/Quadis.mp4` is 5.2 MB and `autoPlay` starts fetching it immediately,
 * which makes it by far the largest thing the site asks anyone to download —
 * for a decorative background, above the copy they actually came for. On a
 * phone on mobile data that is the whole of the mobile performance problem in
 * one file. The poster it already falls back to is 176 KB, i.e. ~3% of it.
 *
 * The connection hints are Chromium-only, so they are read defensively and are
 * never the only signal — the width query is what carries this on iOS.
 */
function heroVideoUnwanted(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return false
  if (HERO_VIDEO_OFF_QUERIES.some((q) => window.matchMedia(q).matches)) return true
  const conn = (navigator as unknown as { connection?: { saveData?: boolean; effectiveType?: string } }).connection
  if (conn?.saveData) return true
  if (conn?.effectiveType && /^(slow-2g|2g|3g)$/.test(conn.effectiveType)) return true
  return false
}

/**
 * HeroVideoShowcase — Full-screen looping background video for the top banner (§1).
 * Features smooth autoPlay loop and high-res poster fallback.
 */
export function HeroVideoShowcase({
  videoUrl = '/videos/Quadis.mp4',
  posterUrl = '/images/home/hero.webp'
}: HeroVideoShowcaseProps) {
  const [videoFailed, setVideoFailed] = useState(false)
  /* Computed in the initialiser, NOT in an effect. The previous reduced-motion
     check started `false` and flipped in useEffect, which runs after the first
     commit — by then `<video autoPlay>` is mounted and the 5.2 MB fetch is
     already in flight, so the fallback saved nothing it was meant to save.
     Reading matchMedia synchronously means the element is never created. */
  const [skipVideo, setSkipVideo] = useState(heroVideoUnwanted)

  useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return
    const mqls = HERO_VIDEO_OFF_QUERIES.map((q) => window.matchMedia(q))
    /* One-way latch. A phone held in landscape is ~844px wide, so re-testing on
       rotation would start the very download this exists to prevent; once we
       have decided against the video for a session we stay decided. The
       listener therefore only ever turns it off — for a desktop window being
       narrowed, where swapping to the still is free. */
    const sync = () => setSkipVideo((prev) => prev || heroVideoUnwanted())
    mqls.forEach((m) => m.addEventListener('change', sync))
    return () => mqls.forEach((m) => m.removeEventListener('change', sync))
  }, [])

  if (videoFailed || skipVideo) {
    /* This is the LCP element whenever it renders, so it is fetched at high
       priority rather than being discovered at normal priority mid-parse. */
    return <img className="hero-media" src={posterUrl} alt="Quadis Hotel Showcase" fetchPriority="high" style={{ objectFit: 'cover', width: '100%', height: '100%', position: 'absolute', inset: 0 }} />
  }

  return (
    <video
      className="hero-media"
      autoPlay
      loop
      muted
      playsInline
      poster={posterUrl}
      onError={() => setVideoFailed(true)}
      style={{ objectFit: 'cover', width: '100%', height: '100%', position: 'absolute', inset: 0 }}
    >
      <source src={videoUrl} type="video/mp4" />
      <img className="hero-media" src={posterUrl} alt="Quadis Hotel Showcase" style={{ objectFit: 'cover', width: '100%', height: '100%', position: 'absolute', inset: 0 }} />
    </video>
  )
}

